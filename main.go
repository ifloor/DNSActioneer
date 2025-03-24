package main

import (
	"dnsactioneer/business"
	"dnsactioneer/model"
	"dnsactioneer/parser"
	"dnsactioneer/utils"
	"log"
)

func main() {
	configuration, err := parser.GetConfiguration()
	if err != nil {
		log.Println("Error getting configurations")
		panic(err)
	}

	businessConfiguration := mapToBusiness(configuration)

	doToken := utils.GetEnvDOToken()
	if doToken == "" {
		panic("DO_TOKEN environment variable is not set. Cannot proceed")
	}

	business.RunActioneerForever(businessConfiguration, doToken)
}

func mapToBusiness(configuration model.WorkConfiguration) business.WorkConfiguration {
	var businessConfiguration business.WorkConfiguration

	var ipConfigurations []business.IPConfiguration
	for _, ipConfig := range configuration.IpIngressBasedOnIpEgress {
		ipConfigurations = append(ipConfigurations, business.IPConfiguration{
			IsGenericRule: ipConfig.IfEgress == "*",
			IfEgressIP:    ipConfig.IfEgress,
			ThenIngressIP: ipConfig.ThenIngress,
		})
	}
	changeTheseDNSs := make(map[string]string)
	for _, dns := range configuration.ChangeTheseDNSs {
		changeTheseDNSs[dns] = dns
	}

	doNotChangeTheseDNSs := make(map[string]string)
	for _, dns := range configuration.DoNotChangeTheseDNSs {
		doNotChangeTheseDNSs[dns] = dns
	}

	businessConfiguration = business.WorkConfiguration{
		IpIngressBasedOnIpEgress: ipConfigurations,
		ChangeTheseDNSs:          changeTheseDNSs,
		DoNotChangeTheseDNSs:     doNotChangeTheseDNSs,
	}

	return businessConfiguration
}
