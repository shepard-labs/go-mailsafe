package emailverifier

import (
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

type Syntax struct {
	Username string
	Domain   string
	Valid    bool
}

func (v *Verifier) ParseAddress(email string) Syntax {
	email = strings.TrimSpace(email)
	if email == "" {
		return Syntax{Valid: false}
	}

	at := strings.LastIndex(email, "@")
	if at < 1 || at == len(email)-1 {
		return Syntax{Valid: false}
	}

	username := email[:at]
	domain := email[at+1:]

	valid := emailRegex.MatchString(email)

	return Syntax{
		Username: username,
		Domain:   strings.ToLower(domain),
		Valid:    valid,
	}
}

func IsAddressValid(email string) bool {
	return emailRegex.MatchString(strings.TrimSpace(email))
}
