package business

import (
	"context"
	"dnsactioneer/utils"
	"errors"
	"github.com/digitalocean/godo"
	"log"
	"time"
)

const recordsPerPage = 1000

type Actioneer struct {
	configuration   WorkConfiguration
	lastEgressIP    string
	doClient        *godo.Client
	trackingRecords []TrackingDomainRecord
}

func RunActioneerForever(configuration WorkConfiguration, doToken string) {
	actioneer := Actioneer{
		configuration: configuration,
		doClient:      godo.NewFromToken(doToken),
	}

	log.Println("Starting to run actioneer forever")
	loopIntervalSeconds := utils.GetEnvLoopIntervalSeconds()
	for {
		log.Println("Running actioneer loop")

		startMillis := time.Now().UnixNano() / int64(time.Millisecond)
		actioneer.loopRun()
		endMillis := time.Now().UnixNano() / int64(time.Millisecond)

		// Sleep difference
		sleepMillis := int64(loopIntervalSeconds)*1000 - (endMillis - startMillis)
		if sleepMillis < 0 {
			log.Println("Loop took longer than the configurated loop interval seconds: ", loopIntervalSeconds)
			continue
		}

		time.Sleep(time.Duration(sleepMillis) * time.Millisecond)
	}
}

func (a *Actioneer) loopRun() {
	if a.lastEgressIP == "" {
		err := a.firstSpecialRun()
		if err != nil {
			log.Println("Error in first special run. Err: ", err)
		}
		return
	}

	err := a.subsequentRun()
	if err != nil {
		log.Println("Error in subsequent run. Err: ", err)
		return
	}
}

func (a *Actioneer) firstSpecialRun() error {
	{
		ip, err := utils.GetPublicIP()
		if err != nil {
			log.Println("Error getting public IP")
			return err
		}

		a.lastEgressIP = ip
	}

	log.Println("First special run with IP: ", a.lastEgressIP)

	err := a.processWholeFlowOnProperMoment()
	if err != nil {
		log.Println("Error processing whole flow on proper moment. Err: ", err)
		return err
	}
	return nil
}

func (a *Actioneer) subsequentRun() error {
	ip, err := utils.GetPublicIP()
	if err != nil {
		log.Println("Error getting public IP for subsequent run")
		return err
	}

	if a.lastEgressIP == ip {
		log.Println("No change in egress IP. Nothing to do then")
		return nil
	}

	// Ip changed!
	log.Println("Egress IP changed from: ", a.lastEgressIP, " to: ", ip)
	a.lastEgressIP = ip
	err = a.processWholeFlowOnProperMoment()
	if err != nil {
		return err
	}

	return nil
}

func (a *Actioneer) processWholeFlowOnProperMoment() error {
	ruleToApply := a.getApplyingRule(a.lastEgressIP)
	if ruleToApply == nil {
		log.Println("No rule to apply that matches the egress IP: ", a.lastEgressIP, ", and nothing can be done then")
		return errors.New("no rule to apply that matches the egress IP")
	}

	err := a.fetchDomainRecords()
	if err != nil {
		return err
	}

	// Check if records are properly configured
	err = a.checkIfDomainRecordsAreCorrect(*ruleToApply)
	if err != nil {
		log.Println("Error checking if domain records are correct. Err: ", err)
		return err
	}

	return nil
}

func (a *Actioneer) fetchDomainRecords() error {
	domains, response, err := a.doClient.Domains.List(context.Background(), nil)
	if err != nil {
		log.Println("Error getting domains. Err: ", err)
		return err
	}

	for _, domain := range domains {
		log.Println("Processing domain: ", domain.Name)

		allRecords, err := a.getAllRecordsForDomain(domain)

		if err != nil {
			log.Println("Error getting domain records. Err: ", err)
			return err
		}

		for _, record := range allRecords {
			if record.Type != "A" { // Process only A records
				continue
			}
			processedRecord := TrackingDomainRecord{
				ForDomain: domain,
				ForRecord: record,
			}
			a.trackingRecords = append(a.trackingRecords, processedRecord)
			log.Println("Identified A DNS record: name: ", record.Name, " type: ", record.Type, " data: ", record.Data, "ID: ", record.ID, " ttl: ", record.TTL, " priority: ", record.Priority, " flags: ", record.Flags, " tag: ", record.Tag, " port: ", record.Port, " weight: ", record.Weight)
		}
	}

	log.Println("rate: ", response.Rate.Remaining, "/", response.Rate.Limit, " reset: ", response.Rate.Reset)

	return nil
}

func (a *Actioneer) checkIfDomainRecordsAreCorrect(ruleToApply IPConfiguration) error {
	thenIP := ruleToApply.ThenIngressIP
	for _, trackingRecord := range a.trackingRecords {
		fullDomain := getFullDomainName(trackingRecord.ForDomain, trackingRecord.ForRecord)

		if a.configuration.ChangeTheseDNSs[fullDomain] != "" {
			log.Println("Record is configured to be analyzed: ", fullDomain)
			if trackingRecord.ForRecord.Data == thenIP {
				log.Println("Record is already configured correctly. Skipping")
				continue
			}

			log.Println("Record ", fullDomain, "is not correct (", trackingRecord.ForRecord.Data, "). Changing it to: ", thenIP)
			err := a.updateRecordToIp(trackingRecord, thenIP)
			if err != nil {
				log.Println("Error updating record. Err: ", err)
				return err
			}
		} else if a.configuration.DoNotChangeTheseDNSs[fullDomain] != "" {
			// Ok, only ignore
			log.Println("Ignoring record (as it was present on 'doNotChange' list): ", fullDomain)
		} else {
			log.Println("A DNS domain is not on 'change' or 'doNotChange' lists. It should not happen: ", fullDomain)
			utils.SendNotification("A DNS domain is not on 'change' or 'doNotChange' lists. It should not happen: " + fullDomain)
		}
	}

	return nil
}

func (a *Actioneer) getApplyingRule(egressIP string) *IPConfiguration {
	for _, ipConfig := range a.configuration.IpIngressBasedOnIpEgress {
		if ipConfig.IsGenericRule || ipConfig.IfEgressIP == egressIP {
			return &ipConfig
		}
	}

	return nil
}

func (a *Actioneer) getAllRecordsForDomain(domain godo.Domain) ([]godo.DomainRecord, error) {
	var returningRecords []godo.DomainRecord

	var records []godo.DomainRecord
	firstFetch := true

	for {
		if !firstFetch && len(records) < recordsPerPage {
			break
		}
		firstFetch = false

		records, _, err := a.doClient.Domains.Records(context.Background(), domain.Name, &godo.ListOptions{
			Page:         0,
			PerPage:      recordsPerPage,
			WithProjects: false,
		})
		if err != nil {
			return nil, err
		}

		returningRecords = append(returningRecords, records...)
	}

	return returningRecords, nil
}

func (a *Actioneer) updateRecordToIp(record TrackingDomainRecord, ip string) error {
	editedRecord, _, err := a.doClient.Domains.EditRecord(context.Background(), record.ForDomain.Name, record.ForRecord.ID, &godo.DomainRecordEditRequest{
		Data: ip,
	})
	if err != nil {
		return err
	}

	log.Println("Updated record: name: ", editedRecord.Name, " type: ", editedRecord.Type, " data: ", editedRecord.Data, "ID: ", editedRecord.ID, " ttl: ", editedRecord.TTL, " priority: ", editedRecord.Priority, " flags: ", editedRecord.Flags, " tag: ", editedRecord.Tag, " port: ", editedRecord.Port, " weight: ", editedRecord.Weight)

	return nil
}

func getFullDomainName(domain godo.Domain, record godo.DomainRecord) string {
	if record.Name == "@" {
		return domain.Name
	}

	return record.Name + "." + domain.Name
}
