package emailverifier

import "strings"

func (v *Verifier) IsFreeDomain(domain string) bool {
	return freeDomainList[strings.ToLower(domain)]
}
