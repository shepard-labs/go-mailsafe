# go-mailsafe

A Go library for comprehensive email address verification **without sending any emails**. Validate syntax, check MX records, detect disposable and free email providers, identify role-based accounts, and suggest corrections for misspelled domains — all from a single `Verify()` call.

All structured output is defined via Protocol Buffers and serialized with `protojson`, giving you stable, well-typed contracts suitable for both library use and HTTP APIs.

---

## Features

| Feature                        | Default   | Description                                                                                            |
|--------------------------------|-----------|--------------------------------------------------------------------------------------------------------|
| **Syntax Validation**          | Always on | RFC 5321/5322-compatible parsing of the local part and domain                                          |
| **MX Record Lookup**           | Always on | DNS MX query to confirm the domain can receive email                                                   |
| **Disposable Email Detection** | Always on | Checks against 5,000+ known disposable email providers                                                 |
| **Free Provider Detection**    | Always on | Checks against 4,700+ known free email provider domains                                                |
| **Role Account Detection**     | Always on | Identifies functional addresses like `admin@`, `support@`, `noreply@` across 880+ known role usernames |
| **Domain Typo Suggestion**     | Opt-in    | Levenshtein-based "did you mean?" correction (`gmial.com` → `gmail.com`)                               |
| **Proto-first Serialization**  | Built-in  | All responses map to protobuf messages; JSON via `protojson`                                           |
| **Self-Hosted API Server**     | Included  | Ready-to-deploy HTTP server at `cmd/apiserver`                                                         |

---

## Installation

### Library

```bash
go get github.com/shepard-labs/go-mailsafe
```

### API Server

```bash
go install github.com/shepard-labs/go-mailsafe/cmd/apiserver@latest
```

### Requirements

- Go 1.25 or later
- `protoc` and `protoc-gen-go` (only if you need to regenerate proto files)

---

## Usage

### Quick Start

```go
package main

import (
	"fmt"
	"log"

	emailverifier "github.com/shepard-labs/go-mailsafe"
)

func main() {
	verifier := emailverifier.NewVerifier()

	result, err := verifier.Verify("user@gmail.com")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Valid:       %t\n", result.Syntax.Valid)
	fmt.Printf("Username:    %s\n", result.Syntax.Username)
	fmt.Printf("Domain:      %s\n", result.Syntax.Domain)
	fmt.Printf("Has MX:      %t\n", result.HasMxRecords)
	fmt.Printf("Disposable:  %t\n", result.Disposable)
	fmt.Printf("Free:        %t\n", result.Free)
	fmt.Printf("Role:        %t\n", result.RoleAccount)
}
```

Output:

```
Valid:       true
Username:    user
Domain:      gmail.com
Has MX:      true
Disposable:  false
Free:        true
Role:        false
```

### Verifier Configuration

The `Verifier` is created once at startup and reused. It is safe for concurrent use. Configuration uses a fluent builder pattern:

```go
verifier := emailverifier.
	NewVerifier().
	EnableDomainSuggest().
	AddDisposableDomains([]string{"internal-temp.io", "throwaway.dev"})
```

| Method                           | Description                                    |
|----------------------------------|------------------------------------------------|
| `NewVerifier()`                  | Create a new verifier with default settings    |
| `EnableDomainSuggest()`          | Turn on domain typo suggestions                |
| `DisableDomainSuggest()`         | Turn off domain typo suggestions (default)     |
| `AddDisposableDomains([]string)` | Add custom domains to the disposable blocklist |

### Syntax Validation

Parse and validate an email address structure:

```go
syntax := verifier.ParseAddress("user@example.com")
fmt.Println(syntax.Valid)    // true
fmt.Println(syntax.Username) // "user"
fmt.Println(syntax.Domain)   // "example.com"
```

For a lightweight boolean-only check without decomposition:

```go
ok := emailverifier.IsAddressValid("user@example.com") // true
ok = emailverifier.IsAddressValid("not-an-email")       // false
```

Invalid syntax is **not** a Go error — `ParseAddress` always returns a populated `Syntax` struct. Check `syntax.Valid` to determine validity.

### MX Record Lookup

Query DNS MX records for a domain:

```go
mx, err := verifier.CheckMX("gmail.com")
if err != nil {
	// Handle DNS failure (see Error Handling below)
}

fmt.Println(mx.HasMXRecord) // true
for _, record := range mx.Records {
	fmt.Printf("  %s (priority %d)\n", record.Host, record.Pref)
}
```

Records are returned sorted by preference (lowest value = highest priority). A domain with no MX records returns `HasMXRecord: false` with a `nil` error — this is a data signal, not a failure.

### Disposable Email Detection

Check if a domain belongs to a known disposable email provider:

```go
verifier.IsDisposable("mailinator.com") // true
verifier.IsDisposable("gmail.com")      // false
```

The built-in list contains 5,000+ domains. Add your own at startup:

```go
verifier.AddDisposableDomains([]string{"custom-throwaway.io"})
verifier.IsDisposable("custom-throwaway.io") // true
```

### Free Provider Detection

Check if a domain is a known free email provider:

```go
verifier.IsFreeDomain("gmail.com")     // true
verifier.IsFreeDomain("company.com")   // false
```

The built-in list contains 4,700+ domains sourced from [free-email-domains](https://github.com/Kikobeats/free-email-domains), covering Gmail, Yahoo, Outlook/Hotmail, ProtonMail, iCloud, AOL, Yandex, Mail.ru, GMX, Tutanota, Fastmail, QQ, 163, Naver, and thousands more.

### Role Account Detection

Check if the username is a well-known functional/role address:

```go
verifier.IsRoleAccount("admin")     // true
verifier.IsRoleAccount("postmaster") // true
verifier.IsRoleAccount("john")      // false
```

Recognized role usernames include: `abuse`, `admin`, `billing`, `contact`, `help`, `hostmaster`, `info`, `marketing`, `no-reply`, `noreply`, `office`, `postmaster`, `press`, `root`, `sales`, `security`, `support`, `sysadmin`, `webmaster`, and 880+ more. The list is sourced from [AfterShip's email-verifier](https://github.com/AfterShip/email-verifier).

### Domain Typo Suggestions

Detect common misspellings in the domain part and suggest corrections:

```go
verifier := emailverifier.NewVerifier().EnableDomainSuggest()

result, _ := verifier.Verify("user@gmial.com")
fmt.Println(result.Suggestion) // "gmail.com"

result, _ = verifier.Verify("user@gmail.com")
fmt.Println(result.Suggestion) // "" (no typo detected)
```

`SuggestDomain` can also be called independently for real-time "did you mean?" UI flows:

```go
verifier.SuggestDomain("yaho.com")   // "yahoo.com"
verifier.SuggestDomain("outlook.com") // "" (exact match, no suggestion)
```

Domain suggestion is **off by default**. Enable it globally with `EnableDomainSuggest()` on the verifier.

### Unified Verify

`Verify()` orchestrates all enabled checks in a single call:

```go
result, err := verifier.Verify("admin@mailinator.com")
if err != nil {
	log.Fatal(err)
}

fmt.Println(result.Email)        // "admin@mailinator.com"
fmt.Println(result.Syntax.Valid) // true
fmt.Println(result.HasMxRecords) // true
fmt.Println(result.Disposable)   // true
fmt.Println(result.RoleAccount)  // true
fmt.Println(result.Free)         // false
fmt.Println(result.Suggestion)   // "" (disabled by default)
```

`Verify()` returns a Go `error` only when the call itself cannot proceed. Negative results (invalid syntax, no MX records, disposable domain) are **not** errors — they populate the response fields with a `nil` error.

### Error Handling

DNS lookup failures are represented as `*LookupError`:

```go
mx, err := verifier.CheckMX("nonexistent-domain.example")
if err != nil {
	var lookupErr *emailverifier.LookupError
	if errors.As(err, &lookupErr) {
		fmt.Println(lookupErr.Message) // human-readable, safe to surface
		fmt.Println(lookupErr.Details) // raw DNS detail, internal use only
	}
}
```

Error constants:

| Constant               | Value                                               | When                              |
|------------------------|-----------------------------------------------------|-----------------------------------|
| `ErrTimeout`           | `"The connection to the mail server has timed out"` | DNS lookup timed out              |
| `ErrNoSuchHost`        | `"Mail server does not exist"`                      | Domain has no DNS entry           |
| `ErrServerUnavailable` | `"Mail server is unavailable"`                      | DNS server returned a failure     |
| `ErrBlocked`           | `"Blocked by mail server"`                          | Request rejected at network level |

DNS failures are transient. Log them and return a gracefully degraded response rather than a 500 error. Never expose `LookupError.Details` to end users.

### Protobuf Serialization

All response types map to protobuf messages defined in `proto/v1/emailverifier.proto`. JSON serialization uses `protojson` exclusively:

```go
import (
	"google.golang.org/protobuf/encoding/protojson"
	emailverifierv1 "github.com/shepard-labs/go-mailsafe/proto/v1"
)

marshaler := protojson.MarshalOptions{
	UseProtoNames:   true,  // snake_case keys matching proto field names
	EmitUnpopulated: false, // omit zero-value fields
}

response := &emailverifierv1.VerifyResponse{
	Email: "user@example.com",
	Syntax: &emailverifierv1.Syntax{
		Username: "user",
		Domain:   "example.com",
		Valid:    true,
	},
	Free: true,
}

jsonBytes, err := marshaler.Marshal(response)
```

Produces:

```json
{
  "email": "user@example.com",
  "syntax": {
    "username": "user",
    "domain": "example.com",
    "valid": true
  },
  "free": true
}
```

### Self-Hosted API Server

The repository includes a ready-to-deploy HTTP API server.

**Build and run:**

```bash
go build -o mailsafe-server ./cmd/apiserver
./mailsafe-server
```

The server listens on port `8080` by default. Set the `PORT` environment variable to override:

```bash
PORT=3000 ./mailsafe-server
```

**Endpoint:**

```
GET /v1/{email}/verify
```

**Example request:**

```bash
curl http://localhost:8080/v1/user@gmail.com/verify
```

**Example response:**

```json
{
  "email": "user@gmail.com",
  "syntax": {
    "username": "user",
    "domain": "gmail.com",
    "valid": true
  },
  "mx": {
    "has_mx_record": true,
    "records": [
      { "host": "gmail-smtp-in.l.google.com.", "pref": 5 },
      { "host": "alt1.gmail-smtp-in.l.google.com.", "pref": 10 },
      { "host": "alt2.gmail-smtp-in.l.google.com.", "pref": 20 }
    ]
  },
  "free": true
}
```

Fields with zero/false/empty values are omitted from the JSON output. The `suggestion` field only appears when domain suggestion is enabled and a typo is detected.

> **Note:** The `cmd/apiserver` binary is a reference implementation. Add rate limiting, authentication, and observability before exposing it externally.

---

## Project Structure

```
github.com/shepard-labs/go-mailsafe/
├── cmd/
│   └── apiserver/
│       └── main.go              # Self-hosted HTTP API server (D-11)
├── proto/
│   └── v1/
│       ├── emailverifier.proto  # Protobuf schema definitions (D-1)
│       └── emailverifier.pb.go  # Generated Go protobuf code
├── verifier.go                  # Verifier constructor and configuration (D-2)
├── syntax.go                    # Email syntax parsing and validation (D-3)
├── mx.go                        # DNS MX record lookup (D-4)
├── disposable.go                # Disposable email detection logic (D-5)
├── free.go                      # Free provider detection logic (D-6)
├── role.go                      # Role account detection logic (D-7)
├── suggestion.go                # Domain typo suggestion engine (D-8)
├── verify.go                    # Unified Verify() entry point (D-9)
├── errors.go                    # LookupError type and DNS error parsing (D-10)
├── metadata_disposable.go       # 5,000+ disposable domain list
├── metadata_free.go             # 4,700+ free email provider domain list
├── metadata_role.go             # 880+ role-based username list
├── metadata_suggestion.go       # Common domains for typo detection
├── verifier_test.go             # Test suite
├── go.mod
└── go.sum
```

---

## FAQ

### Does this library send any emails?

No. All checks are performed locally or via DNS lookups. No SMTP connections are made and no emails are sent at any point.

### Is the Verifier safe for concurrent use?

Yes. A single `Verifier` instance should be created at startup and shared across goroutines. The disposable domain map is populated at construction time and is read-only during verification. The only mutation methods (`AddDisposableDomains`, `EnableDomainSuggest`, `DisableDomainSuggest`) are intended for use during initialization, before concurrent access begins.

### Why does `Verify()` return a result instead of an error for invalid emails?

By design, a syntactically invalid email, a domain with no MX records, or a disposable address are all **data outcomes**, not exceptional conditions. The `error` return is reserved for cases where the verification process itself cannot proceed. This lets callers make policy decisions (e.g., reject disposable addresses) without error-handling boilerplate.

### Why `protojson` instead of `encoding/json`?

`protojson` ensures the JSON output is always consistent with the proto schema — field names, casing, and zero-value omission behavior are all governed by the `.proto` file. This avoids drift between your proto contracts and your JSON API surface.

### How do I update the disposable domain list?

The disposable domain list in `metadata_disposable.go` is sourced from [disposable-email-domains](https://github.com/disposable-email-domains/disposable-email-domains). To update it, fetch the latest blocklist and regenerate the file:

```bash
curl -sL "https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/master/disposable_email_blocklist.conf" -o /tmp/disposable_domains.txt
```

Then regenerate `metadata_disposable.go` with the domain list as a Go string slice.

For runtime additions without rebuilding, use `AddDisposableDomains()` at startup.

### Can I use individual checks without calling `Verify()`?

Yes. Every check is exposed as a standalone method on the `Verifier`:

```go
verifier.ParseAddress(email)       // syntax only
verifier.CheckMX(domain)           // MX only
verifier.IsDisposable(domain)      // disposable check only
verifier.IsFreeDomain(domain)      // free provider check only
verifier.IsRoleAccount(username)   // role account check only
verifier.SuggestDomain(domain)     // domain typo check only
```

### What happens when DNS lookups fail?

DNS failures are returned as `*LookupError` with a human-readable `Message` and a raw `Details` string. The `Verify()` method handles these gracefully — it sets `HasMxRecords` to `false` and continues with the remaining checks rather than aborting.

### Can I use this behind a corporate firewall or air-gapped environment?

Yes. All detection lists (disposable, free, role, suggestion) are compiled into the binary as static Go data. No network calls are required except for DNS MX lookups, which use your system's configured DNS resolver.

---

## Credits

This project is built with the help of the following open source projects:

- **[disposable-email-domains](https://github.com/disposable-email-domains/disposable-email-domains)** — Community-maintained list of disposable email address domains. Used as the upstream source for the disposable email blocklist.
- **[free-email-domains](https://github.com/Kikobeats/free-email-domains)** — Comprehensive list of free email provider domains. Used as the upstream source for free provider detection.
- **[Protocol Buffers (protobuf)](https://github.com/protocolbuffers/protobuf)** — Google's language-neutral, platform-neutral serialization mechanism. Used to define all data contracts.
- **[protobuf-go](https://github.com/protocolbuffers/protobuf-go)** (`google.golang.org/protobuf`) — The Go implementation of Protocol Buffers, including the `protojson` package used for JSON serialization.
- **[chi](https://github.com/go-chi/chi)** (`github.com/go-chi/chi/v5`) — Lightweight, idiomatic HTTP router for Go. Used in the self-hosted API server.

---

## Contributing

We welcome contributions of all kinds — bug fixes, new features, documentation improvements, and dataset updates.

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines on how to contribute, including code style, testing requirements, and the pull request process.

---

## License

This project is licensed under the **GNU General Public License v3.0**. See [LICENSE](LICENSE) for the full license text.
