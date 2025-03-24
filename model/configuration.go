package model

type WorkConfiguration struct {
	IpIngressBasedOnIpEgress []IPConfiguration `json:"ipIngressBasedOnIpEgress"`
	ChangeTheseDNSs          []string          `json:"changeTheseDNSs"`
	DoNotChangeTheseDNSs     []string          `json:"doNotChangeTheseDNSs"`
}

type IPConfiguration struct {
	IfEgress    string `json:"ifEgress"`
	ThenIngress string `json:"thenIngress"`
}
