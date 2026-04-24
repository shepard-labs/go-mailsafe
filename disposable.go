package emailverifier

import "strings"

func (v *Verifier) IsDisposable(domain string) bool {
	return v.disposableDomains[strings.ToLower(domain)]
}
