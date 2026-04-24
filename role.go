package emailverifier

import "strings"

func (v *Verifier) IsRoleAccount(username string) bool {
	return roleAccountList[strings.ToLower(username)]
}
