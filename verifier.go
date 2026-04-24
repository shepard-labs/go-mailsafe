package emailverifier

type Verifier struct {
	domainSuggestEnabled bool
	disposableDomains    map[string]bool
}

func NewVerifier() *Verifier {
	v := &Verifier{
		disposableDomains: make(map[string]bool),
	}
	for _, d := range disposableDomainList {
		v.disposableDomains[d] = true
	}
	return v
}

func (v *Verifier) EnableDomainSuggest() *Verifier {
	v.domainSuggestEnabled = true
	return v
}

func (v *Verifier) DisableDomainSuggest() *Verifier {
	v.domainSuggestEnabled = false
	return v
}

func (v *Verifier) AddDisposableDomains(domains []string) *Verifier {
	for _, d := range domains {
		v.disposableDomains[d] = true
	}
	return v
}
