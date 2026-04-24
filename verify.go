package emailverifier

type Result struct {
	Email        string
	Syntax       Syntax
	HasMxRecords bool
	Disposable   bool
	RoleAccount  bool
	Free         bool
	Suggestion   string
}

func (v *Verifier) Verify(email string) (*Result, error) {
	syntax := v.ParseAddress(email)

	result := &Result{
		Email:  email,
		Syntax: syntax,
	}

	if !syntax.Valid {
		return result, nil
	}

	domain := syntax.Domain
	username := syntax.Username

	mx, err := v.CheckMX(domain)
	if err != nil {
		result.HasMxRecords = false
	} else {
		result.HasMxRecords = mx.HasMXRecord
	}

	result.Disposable = v.IsDisposable(domain)
	result.Free = v.IsFreeDomain(domain)
	result.RoleAccount = v.IsRoleAccount(username)

	if v.domainSuggestEnabled {
		result.Suggestion = v.SuggestDomain(domain)
	}

	return result, nil
}
