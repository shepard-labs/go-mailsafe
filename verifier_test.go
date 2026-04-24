package emailverifier

import (
	"testing"
)

func TestParseAddress_Valid(t *testing.T) {
	v := NewVerifier()
	s := v.ParseAddress("user@example.com")
	if !s.Valid {
		t.Fatal("expected valid")
	}
	if s.Username != "user" {
		t.Fatalf("expected username 'user', got %q", s.Username)
	}
	if s.Domain != "example.com" {
		t.Fatalf("expected domain 'example.com', got %q", s.Domain)
	}
}

func TestParseAddress_Invalid(t *testing.T) {
	v := NewVerifier()
	cases := []string{"", "noatsign", "@nodomain", "user@", "@@", " "}
	for _, c := range cases {
		s := v.ParseAddress(c)
		if s.Valid {
			t.Fatalf("expected invalid for %q", c)
		}
	}
}

func TestIsAddressValid(t *testing.T) {
	if !IsAddressValid("test@example.com") {
		t.Fatal("expected valid")
	}
	if IsAddressValid("invalid") {
		t.Fatal("expected invalid")
	}
}

func TestIsDisposable(t *testing.T) {
	v := NewVerifier()
	if !v.IsDisposable("0-mail.com") {
		t.Fatal("expected disposable")
	}
	if v.IsDisposable("gmail.com") {
		t.Fatal("gmail should not be disposable")
	}
}

func TestAddDisposableDomains(t *testing.T) {
	v := NewVerifier().AddDisposableDomains([]string{"custom-temp.io"})
	if !v.IsDisposable("custom-temp.io") {
		t.Fatal("expected custom domain to be disposable")
	}
}

func TestIsFreeDomain(t *testing.T) {
	v := NewVerifier()
	if !v.IsFreeDomain("gmail.com") {
		t.Fatal("expected gmail to be free")
	}
	if v.IsFreeDomain("example.com") {
		t.Fatal("example.com should not be free")
	}
}

func TestIsRoleAccount(t *testing.T) {
	v := NewVerifier()
	if !v.IsRoleAccount("admin") {
		t.Fatal("expected admin to be role account")
	}
	if !v.IsRoleAccount("postmaster") {
		t.Fatal("expected postmaster to be role account")
	}
	if v.IsRoleAccount("john") {
		t.Fatal("john should not be role account")
	}
}

func TestSuggestDomain(t *testing.T) {
	v := NewVerifier()
	s := v.SuggestDomain("gmial.com")
	if s != "gmail.com" {
		t.Fatalf("expected 'gmail.com', got %q", s)
	}
	s = v.SuggestDomain("gmail.com")
	if s != "" {
		t.Fatalf("expected empty suggestion for exact match, got %q", s)
	}
}

func TestVerify_InvalidSyntax(t *testing.T) {
	v := NewVerifier()
	result, err := v.Verify("notanemail")
	if err != nil {
		t.Fatal(err)
	}
	if result.Syntax.Valid {
		t.Fatal("expected invalid syntax")
	}
}

func TestVerify_RoleAndFree(t *testing.T) {
	v := NewVerifier()
	result, err := v.Verify("admin@gmail.com")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Syntax.Valid {
		t.Fatal("expected valid syntax")
	}
	if !result.RoleAccount {
		t.Fatal("expected role account")
	}
	if !result.Free {
		t.Fatal("expected free domain")
	}
}

func TestDomainSuggestToggle(t *testing.T) {
	v := NewVerifier()
	result, _ := v.Verify("user@gmial.com")
	if result.Suggestion != "" {
		t.Fatal("suggestion should be empty when disabled")
	}

	v.EnableDomainSuggest()
	result, _ = v.Verify("user@gmial.com")
	if result.Suggestion != "gmail.com" {
		t.Fatalf("expected 'gmail.com' suggestion, got %q", result.Suggestion)
	}

	v.DisableDomainSuggest()
	result, _ = v.Verify("user@gmial.com")
	if result.Suggestion != "" {
		t.Fatal("suggestion should be empty after disable")
	}
}

func TestLookupError(t *testing.T) {
	le := &LookupError{Message: ErrTimeout, Details: "context deadline exceeded"}
	if le.Error() != ErrTimeout {
		t.Fatalf("expected %q, got %q", ErrTimeout, le.Error())
	}
}
