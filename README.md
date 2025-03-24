# Description
Tool that keeps running and check if the egress public IP of a private network changed. 
When it changes it updates the A DNS records on DigitalOcean DNS system. These domains can be pre-configured in the configs with the IP configured on the
egress/ingress selection rules.

# Sample configuration
```json
{
  "ipIngressBasedOnIpEgress":
  [
    {
      "ifEgress": "177.27.236.195",
      "thenIngress": "177.27.236.195"
    },
    {
      "ifEgress": "*",
      "thenIngress": "165.227.19.132"
    }
  ],
  "changeTheseDNSs":
  [
    "domain.that.will.be.changed.com"
  ],
  "doNotChangeTheseDNSs":
  [
    "domain.that.will.not.be.changed.com"
  ]
}
```