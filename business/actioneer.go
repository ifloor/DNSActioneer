package business

import (
	"context"
	"dnsactioneer/utils"
	"errors"
	"github.com/digitalocean/godo"
	"github.com/rs/zerolog/log"
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

	log.Info().Msgf("Starting to run actioneer forever")
	loopIntervalSeconds := utils.GetEnvLoopIntervalSeconds()
	for {
		log.Info().Msgf("Running actioneer loop")

		startMillis := time.Now().UnixNano() / int64(time.Millisecond)
		actioneer.loopRun()
		endMillis := time.Now().UnixNano() / int64(time.Millisecond)

		// Sleep difference
		sleepMillis := int64(loopIntervalSeconds)*1000 - (endMillis - startMillis)
		if sleepMillis < 0 {
			log.Info().Msgf("Loop took longer than the configurated loop interval seconds: %v", loopIntervalSeconds)
			continue
		}

		time.Sleep(time.Duration(sleepMillis) * time.Millisecond)
	}
}

func (a *Actioneer) loopRun() {
	if a.lastEgressIP == "" {
		err := a.firstSpecialRun()
		if err != nil {
			log.Info().Msgf("Error in first special run. Err: %v", err)
		}
		return
	}

	err := a.subsequentRun()
	if err != nil {
		log.Info().Msgf("Error in subsequent run. Err: %v", err)
		return
	}
}

func (a *Actioneer) firstSpecialRun() error {
	{
		ip, err := utils.GetPublicIP()
		if err != nil {
			log.Error().Msgf("Error getting public IP")
			return err
		}

		a.lastEgressIP = ip
	}

	log.Info().Msgf("First special run with IP: %v", a.lastEgressIP)

	err := a.processWholeFlowOnProperMoment()
	if err != nil {
		log.Info().Msgf("Error processing whole flow on proper moment. Err: %v", err)
		return err
	}
	return nil
}

func (a *Actioneer) subsequentRun() error {
	ip, err := utils.GetPublicIP()
	if err != nil {
		log.Error().Msgf("Error getting public IP for subsequent run")
		return err
	}

	if a.lastEgressIP == ip {
		log.Info().Msgf("No change in egress IP. Nothing to do then")
		return nil
	}

	// Ip changed!
	log.Info().Msgf("Egress IP changed from: %v to: %v", a.lastEgressIP, ip)
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
		log.Info().Msgf("No rule to apply that matches the egress IP: %v, and nothing can be done then", a.lastEgressIP)
		return errors.New("no rule to apply that matches the egress IP")
	}

	err := a.fetchDomainRecords()
	if err != nil {
		return err
	}

	// Check if records are properly configured
	err = a.checkIfDomainRecordsAreCorrect(*ruleToApply)
	if err != nil {
		log.Info().Msgf("Error checking if domain records are correct. Err: %v", err)
		return err
	}

	return nil
}

func (a *Actioneer) fetchDomainRecords() error {
	domains, response, err := a.doClient.Domains.List(context.Background(), nil)
	if err != nil {
		log.Error().Msgf("Error getting domains. Err: %v", err)
		return err
	}

	for _, domain := range domains {
		log.Info().Msgf("Processing domain: %v", domain.Name)

		allRecords, err := a.getAllRecordsForDomain(domain)

		if err != nil {
			log.Error().Msgf("Error getting domain records. Err: %v", err)
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
			log.Info().Msgf("Identified A DNS record: name: %v type: %v data: %v ID: %v ttl: %v priority: %v flags: %v tag: %v port: %v weight: %v", record.Name, record.Type, record.Data, record.ID, record.TTL, record.Priority, record.Flags, record.Tag, record.Port, record.Weight)
		}
	}

	log.Info().Msgf("rate: %v/%v reset: %v", response.Rate.Remaining, response.Rate.Limit, response.Rate.Reset)

	return nil
}

func (a *Actioneer) checkIfDomainRecordsAreCorrect(ruleToApply IPConfiguration) error {
	thenIP := ruleToApply.ThenIngressIP
	for _, trackingRecord := range a.trackingRecords {
		fullDomain := getFullDomainName(trackingRecord.ForDomain, trackingRecord.ForRecord)

		if a.configuration.ChangeTheseDNSs[fullDomain] != "" {
			log.Info().Msgf("Record is configured to be analyzed: %v", fullDomain)
			if trackingRecord.ForRecord.Data == thenIP {
				log.Info().Msgf("Record is already configured correctly. Skipping")
				continue
			}

			log.Info().Msgf("Record %v is not correct (%v). Changing it to: %v", fullDomain, trackingRecord.ForRecord.Data, thenIP)
			err := a.updateRecordToIp(trackingRecord, thenIP)
			if err != nil {
				log.Info().Msgf("Error updating record. Err: %v", err)
				return err
			}
		} else if a.configuration.DoNotChangeTheseDNSs[fullDomain] != "" {
			// Ok, only ignore
			log.Info().Msgf("Ignoring record (as it was present on 'doNotChange' list): %v", fullDomain)
		} else {
			log.Error().Msgf("A DNS domain is not on 'change' or 'doNotChange' lists. It should not happen: %v", fullDomain)
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

	log.Info().Msgf("Updated record: name: %v type: %v data: %v ID: %v ttl: %v priority: %v flags: %v tag: %v port: %v weight: %v", editedRecord.Name, editedRecord.Type, editedRecord.Data, editedRecord.ID, editedRecord.TTL, editedRecord.Priority, editedRecord.Flags, editedRecord.Tag, editedRecord.Port, editedRecord.Weight)

	return nil
}

func getFullDomainName(domain godo.Domain, record godo.DomainRecord) string {
	if record.Name == "@" {
		return domain.Name
	}

	return record.Name + "." + domain.Name
}
