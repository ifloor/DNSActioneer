package business

import "github.com/digitalocean/godo"

type WorkConfiguration struct {
	IpIngressBasedOnIpEgress []IPConfiguration
	ChangeTheseDNSs          map[string]string
	DoNotChangeTheseDNSs     map[string]string
}

type IPConfiguration struct {
	IsGenericRule bool
	IfEgressIP    string
	ThenIngressIP string
}

//

type TrackingDomainRecord struct {
	ForDomain godo.Domain
	ForRecord godo.DomainRecord
}
