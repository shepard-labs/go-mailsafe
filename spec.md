# email-verifier — Implementation Guide

> **Serialization:** `google.golang.org/protobuf` + `protojson`
> **Disposable domain source:** `github.com/disposable-email-domains/disposable-email-domains`

A Go library for email verification **without sending any emails**. This document is structured as discrete, independently implementable deliverables. Each section can be picked up and implemented by a single engineer.

---

## Deliverable Summary

| # | Deliverable | Default State | Proto Message(s) |
|---|-------------|---------------|-----------------|
| D-1 | Proto Schema & Serialization | — | All messages defined here |
| D-2 | Verifier Initialization & Configuration | — | — |
| D-3 | Syntax Validation | ✅ Always on | `Syntax` |
| D-4 | MX Record Validation | ✅ Always on | `MXResult`, `MXRecord` |
| D-5 | Disposable Email Detection | ✅ Always on | field on `VerifyResponse` |
| D-6 | Free Provider Detection | ✅ Always on | field on `VerifyResponse` |
| D-7 | Role Account Detection | ✅ Always on | field on `VerifyResponse` |
| D-8 | Domain Typo Suggestions | ❌ Opt-in | field on `VerifyResponse` |
| D-9 | Unified Verify Entry Point | ✅ Orchestrates D-3 through D-8 | `VerifyRequest`, `VerifyResponse` |
| D-10 | Error Handling | — | `LookupError` |
| D-11 | Self-Hosted API Server | — | Uses `VerifyResponse` |

---

## D-1: Proto Schema & Serialization

**Purpose:** Define the canonical data contracts for all inputs, outputs, and errors. All Go types returned by the library must map to these proto messages. JSON serialization is done exclusively via `protojson` — never `encoding/json` directly on these types.

### Proto Definition

```proto
syntax = "proto3";

package emailverifier.v1;

option go_package = "github.com/yourorg/emailverifier/proto/v1;emailverifierv1";

// VerifyRequest is the input to the Verify RPC / HTTP handler.
message VerifyRequest {
  string email = 1;

  // Options control which optional checks are activated.
  VerifyOptions options = 2;
}

message VerifyOptions {
  // enable_domain_suggest activates typo detection on the domain portion.
  // Off by default; see D-8.
  bool enable_domain_suggest = 1;

  // extra_disposable_domains appends caller-supplied domains to the
  // built-in disposable list. See D-5.
  repeated string extra_disposable_domains = 2;
}

// VerifyResponse is the unified result of all enabled checks.
message VerifyResponse {
  // The original email address that was verified.
  string email = 1;

  // Syntax contains the parsed components of the address.
  Syntax syntax = 2;

  // mx contains DNS MX record information for the domain.
  MXResult mx = 3;

  // disposable is true when the domain belongs to a known DEA provider.
  bool disposable = 4;

  // role_account is true when the username is a well-known role address.
  bool role_account = 5;

  // free is true when the domain is a known free email provider.
  bool free = 6;

  // suggestion is a typo-corrected domain, populated only when
  // VerifyOptions.enable_domain_suggest is true and a likely typo is found.
  // Empty string means no suggestion.
  string suggestion = 7;
}

// Syntax holds the parsed components and validity of an email address.
message Syntax {
  string username = 1;   // local part before the @
  string domain   = 2;   // domain part after the @
  bool   valid    = 3;   // true if the address passes format validation
}

// MXResult holds the DNS MX lookup outcome for a domain.
message MXResult {
  bool              has_mx_record = 1;
  repeated MXRecord records       = 2;
}

// MXRecord represents a single DNS MX entry.
message MXRecord {
  string host = 1;   // fully-qualified mail host name
  uint32 pref = 2;   // preference (lower = higher priority)
}

// LookupError represents a structured error from a DNS lookup failure.
message LookupError {
  string message = 1;
  string details = 2;
}
```

### Serialization: `protojson`

All JSON marshaling and unmarshaling of these messages **must** use `protojson`, not `encoding/json`.

```go
import "google.golang.org/protobuf/encoding/protojson"

// Marshal to JSON
marshaler := protojson.MarshalOptions{
    UseProtoNames:   true,  // snake_case keys, matching proto field names
    EmitUnpopulated: false, // omit zero-value fields
}
jsonBytes, err := marshaler.Marshal(response)

// Unmarshal from JSON
var req emailverifierv1.VerifyRequest
err := protojson.Unmarshal(jsonBytes, &req)
```

> **Why `UseProtoNames: true`:** By default `protojson` emits camelCase keys (e.g. `hasMxRecord`). Setting `UseProtoNames: true` emits the snake_case names declared in the `.proto` file (e.g. `has_mx_record`), which is consistent with the library's original JSON tags and easier to reason about in logs and APIs.

### Expected JSON Output (canonical shape)

```json
{
  "email": "user@exampledomain.org",
  "syntax": {
    "username": "user",
    "domain": "exampledomain.org",
    "valid": true
  },
  "mx": {
    "has_mx_record": true,
    "records": [
      { "host": "mail.exampledomain.org", "pref": 10 }
    ]
  },
  "disposable": false,
  "role_account": false,
  "free": false,
  "suggestion": ""
}
```

### Field Number Stability

Once the `.proto` file is committed and any wire format or stored data exists, **field numbers must never change**. Add new fields with new numbers; never reuse or remove existing ones.

---

## D-2: Verifier Initialization & Configuration

**Purpose:** Create and configure the single `Verifier` instance used across all checks. In practice, one verifier should be constructed at application startup and reused — it is safe for concurrent use.

### Constructor

```go
verifier := emailverifier.NewVerifier()
```

### Available Configuration After Scoping

The following methods are the **only** configuration options in scope.

| Method | Description |
|--------|-------------|
| `EnableDomainSuggest() *Verifier` | Activate domain typo detection (D-8) |
| `DisableDomainSuggest() *Verifier` | Deactivate domain typo detection (default) |
| `AddDisposableDomains(domains []string) *Verifier` | Inject additional disposable domains at startup (D-5) |

### Recommended Startup Pattern

```go
var verifier = emailverifier.
    NewVerifier().
    EnableDomainSuggest().
    AddDisposableDomains(localCustomDomains)
```

---

## D-3: Syntax Validation

**Purpose:** Parse and validate the structural format of an email address. This check always runs — it cannot be disabled. If syntax is invalid, no further checks are meaningful.

### API

```go
// ParseAddress parses the address into its components and validates format.
func (v *Verifier) ParseAddress(email string) Syntax

// IsAddressValid is a lightweight package-level regex check.
// Use when you only need a valid/invalid answer with no decomposition.
func IsAddressValid(email string) bool
```

### Proto Mapping

The `Syntax` struct maps directly to the `Syntax` proto message:

| Go field   | Proto field | JSON key   |
|------------|-------------|------------|
| `Username` | `username`  | `username` |
| `Domain`   | `domain`    | `domain`   |
| `Valid`    | `valid`     | `valid`    |

### Usage

```go
syntax := verifier.ParseAddress("user@example.com")
// syntax.Valid    → true
// syntax.Username → "user"
// syntax.Domain   → "example.com"

proto := &emailverifierv1.Syntax{
    Username: syntax.Username,
    Domain:   syntax.Domain,
    Valid:    syntax.Valid,
}
```

### Implementation Notes

- `ParseAddress` performs full structural parsing (RFC 5321/5322 compatible).
- `IsAddressValid` is regex-only and faster but less thorough. Use it for pre-validation hot paths.
- A `false` `Valid` field in the response **must not** be treated as an error — it is a valid, populated `Syntax` message with a negative result. Do not return a Go error to callers; return the populated `VerifyResponse` with `syntax.valid = false`.
- When `valid` is `false`, the `username` and `domain` fields may still be partially populated with whatever was parsed before the failure.

---

## D-4: MX Record Validation

**Purpose:** Confirm that the email's domain has at least one DNS MX record, indicating it is configured to receive email. Runs as part of every `Verify()` call.

### API

```go
// CheckMX queries DNS for MX records on the given domain,
// returning them sorted by preference (lowest pref = highest priority).
func (v *Verifier) CheckMX(domain string) (*Mx, error)
```

The library-native `Mx` struct:

```go
type Mx struct {
    HasMXRecord bool       // whether ≥1 MX record exists
    Records     []*net.MX  // the full sorted record list
}
```

### Proto Mapping

| Go field      | Proto message | Proto field      | JSON key         |
|---------------|---------------|------------------|------------------|
| `HasMXRecord` | `MXResult`    | `has_mx_record`  | `has_mx_record`  |
| `Records`     | `MXResult`    | `records`        | `records`        |
| `net.MX.Host` | `MXRecord`    | `host`           | `host`           |
| `net.MX.Pref` | `MXRecord`    | `pref`           | `pref`           |

### Conversion Helper

```go
func toProtoMX(mx *emailverifier.Mx) *emailverifierv1.MXResult {
    if mx == nil {
        return &emailverifierv1.MXResult{}
    }
    result := &emailverifierv1.MXResult{
        HasMxRecord: mx.HasMXRecord,
    }
    for _, r := range mx.Records {
        result.Records = append(result.Records, &emailverifierv1.MXRecord{
            Host: r.Host,
            Pref: uint32(r.Pref),
        })
    }
    return result
}
```

### Error Handling

`CheckMX` returns a `*LookupError` (see D-10) on DNS failure. A domain with no MX records is **not** an error — `CheckMX` returns successfully with `HasMXRecord = false`. Treat a missing MX record as a data signal, not an exception.

### Implementation Notes

- `has_mx_record: false` should not cause a hard rejection on its own; surface it in the response and let the caller decide policy.
- The `records` array will be empty when `has_mx_record` is `false`.
- DNS timeouts surface as `LookupError` with `message = "The connection to the mail server has timed out"`.

---

## D-5: Disposable Email Detection

**Purpose:** Detect whether the email's domain belongs to a known Disposable Email Address (DEA) provider. Uses the community-maintained list at `github.com/disposable-email-domains/disposable-email-domains` as its upstream domain source.

### Upstream Domain Source

The disposable domain list is maintained at:

```
https://github.com/disposable-email-domains/disposable-email-domains
```

The canonical raw file to fetch is:

```
https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/master/disposable_email_blocklist.conf
```

One domain per line, no headers, plain text.

### API

```go
// IsDisposable returns true if the domain is in the disposable list.
func (v *Verifier) IsDisposable(domain string) bool

// AddDisposableDomains merges the provided domains into the active list.
// Call at startup before any verifications begin.
func (v *Verifier) AddDisposableDomains(domains []string) *Verifier
```

### Proto Mapping

The result surfaces as the `disposable` boolean field on `VerifyResponse`:

```proto
bool disposable = 4;
```

### Custom Domain Injection

Operators can supply additional domains (internal blocklists, newly observed DEA providers not yet in the upstream list) via `AddDisposableDomains`. This is additive and persists for the lifetime of the verifier.

```go
custom := []string{"internalspam.io", "tempbox.example.net"}
verifier.AddDisposableDomains(custom)
```

### Implementation Notes

- Pass the **domain** to `IsDisposable`, not the full email address.
- The check is case-insensitive internally; normalize the domain to lowercase before passing.

---

## D-6: Free Provider Detection

**Purpose:** Flag whether the email's domain is a known free email provider (e.g. Gmail, Yahoo Mail, Outlook, ProtonMail). Useful for distinguishing personal from business email addresses.

### API

```go
// IsFreeDomain returns true if the domain is a known free email provider.
func (v *Verifier) IsFreeDomain(domain string) bool
```

### Proto Mapping

Surfaces as the `free` boolean field on `VerifyResponse`:

```proto
bool free = 6;
```

### Implementation Notes

- Pass the **domain**, not the full email address.
- The list is a static bundled dataset (`metadata_free.go`); there is no auto-update mechanism for free provider data.
- `free: true` does not imply the email is invalid or problematic — it is purely informational. Use it in business logic (e.g. requiring a corporate email domain) but do not reject free addresses by default.

---

## D-7: Role Account Detection

**Purpose:** Detect whether the username (local part) of the email address is a well-known role-based or functional account name rather than a personal address.

Common role usernames: `admin`, `info`, `support`, `noreply`, `no-reply`, `postmaster`, `webmaster`, `sales`, `billing`, `contact`, `help`.

### API

```go
// IsRoleAccount returns true if the username is a known role-based name.
func (v *Verifier) IsRoleAccount(username string) bool
```

### Proto Mapping

Surfaces as the `role_account` boolean field on `VerifyResponse`:

```proto
bool role_account = 5;
```

### Implementation Notes

- Pass the **username** (the part before `@`), not the full email address. Extract it from `Syntax.Username` after running D-3.
- The list is a static bundled dataset (`metadata_role.go`).
- `role_account: true` means email sent to this address may be read by multiple people or by automated systems. It is informational — not grounds for rejection unless the use case requires personal addresses.

---

## D-8: Domain Typo Suggestions

**Purpose:** Detect common misspellings in the domain part of an email address (e.g. `gmial.com` → `gmail.com`) and surface a correction for the user.

This check is **disabled by default** in `Verify()` and must be opted in via `EnableDomainSuggest()` on the verifier, or via `VerifyOptions.enable_domain_suggest` on a per-request basis.

### API

```go
// EnableDomainSuggest activates typo checking during Verify().
func (v *Verifier) EnableDomainSuggest() *Verifier

// DisableDomainSuggest deactivates typo checking during Verify(). (default)
func (v *Verifier) DisableDomainSuggest() *Verifier

// SuggestDomain checks a domain in isolation for possible misspellings.
// Returns the suggested correct domain, or an empty string if none found.
func (v *Verifier) SuggestDomain(domain string) string
```

### Proto Mapping

Surfaces as the `suggestion` string field on `VerifyResponse`:

```proto
string suggestion = 7;
```

An empty string means no typo was detected. Do not emit this field in JSON when it is empty (controlled by `EmitUnpopulated: false` in `protojson.MarshalOptions`).

### Standalone Usage

`SuggestDomain` can be called independently of `Verify()`, for example in a real-time "did you mean?" UI flow:

```go
suggestion := verifier.SuggestDomain("gmai.com")
// → "gmail.com"

suggestion = verifier.SuggestDomain("gmail.com")
// → "" (no typo detected)
```

### Implementation Notes

- The suggestion engine uses the bundled `metadata_suggestion.go` dataset (a curated list of common domains with edit-distance comparison).
- Only the **domain** part is checked — the username is never analyzed for typos.
- A non-empty `suggestion` does not mean the original domain is invalid. The domain may still have MX records and be reachable. Surface the suggestion as a UX hint, not a hard failure.
- When used via `Verify()`, ensure `EnableDomainSuggest()` has been called on the verifier **or** `VerifyOptions.enable_domain_suggest` is `true`.

---

## D-9: Unified Verify Entry Point

**Purpose:** Orchestrate all enabled checks in a single call and return a fully populated `VerifyResponse`. This is the primary integration point for application code.

### API

```go
// Verify runs syntax, MX, disposable, free-domain, role-account, and
// (optionally) domain-suggestion checks on the given email address.
func (v *Verifier) Verify(email string) (*Result, error)
```

The library-native `Result` struct:

```go
type Result struct {
    Email        string  // the input email
    Syntax       Syntax  // D-3
    HasMxRecords bool    // D-4 (convenience field; mirrors Mx.HasMXRecord)
    Disposable   bool    // D-5
    RoleAccount  bool    // D-7
    Free         bool    // D-6
    Suggestion   string  // D-8 (empty unless domain suggest is enabled)
}
```

### Mapping `Result` → `VerifyResponse`

```go
func toProtoResponse(r *emailverifier.Result, mx *emailverifierv1.MXResult) *emailverifierv1.VerifyResponse {
    return &emailverifierv1.VerifyResponse{
        Email: r.Email,
        Syntax: &emailverifierv1.Syntax{
            Username: r.Syntax.Username,
            Domain:   r.Syntax.Domain,
            Valid:    r.Syntax.Valid,
        },
        Mx:          mx,           // populated via CheckMX → toProtoMX (D-4)
        Disposable:  r.Disposable,
        RoleAccount: r.RoleAccount,
        Free:        r.Free,
        Suggestion:  r.Suggestion,
    }
}
```

### Full Usage Example

```go
var verifier = emailverifier.
    NewVerifier().
    EnableDomainSuggest().

func VerifyEmail(email string) (*emailverifierv1.VerifyResponse, error) {
    result, err := verifier.Verify(email)
    if err != nil {
        return nil, err
    }

    mx, mxErr := verifier.CheckMX(result.Syntax.Domain)
    if mxErr != nil {
        // DNS failure is a data signal, not a fatal error. See D-10.
        mx = &emailverifier.Mx{HasMXRecord: false}
    }

    return toProtoResponse(result, toProtoMX(mx)), nil
}
```

### Default Check States at `Verify()` Time

| Check | Runs by Default |
|-------|----------------|
| Syntax validation | ✅ Always |
| MX record check | ✅ Always |
| Disposable detection | ✅ Always |
| Free provider detection | ✅ Always |
| Role account detection | ✅ Always |
| Domain typo suggestion | ❌ Requires `EnableDomainSuggest()` |

### Error vs. Negative Result

`Verify()` returns a Go `error` only when the call itself cannot proceed (e.g. the input is entirely unparseable). Checks that yield negative results (invalid syntax, no MX records, disposable domain, etc.) are **not** errors — they populate the response fields and return a `nil` error. Do not conflate the two in error handling.

---

## D-10: Error Handling

**Purpose:** Define how DNS and lookup failures are represented and surfaced in proto responses.

### `LookupError` Type

Used for DNS MX lookup failures. Maps to the `LookupError` proto message.

```go
type LookupError struct {
    Message string // human-readable error category (safe to surface)
    Details string // raw DNS / server response detail (internal use only)
}
```

```proto
message LookupError {
  string message = 1;
  string details = 2;
}
```

### Error Constants (DNS / MX scope)

Only the following constants remain in scope:

| Constant               | Value                                                   | When it occurs                        |
|------------------------|---------------------------------------------------------|---------------------------------------|
| `ErrTimeout`           | `"The connection to the mail server has timed out"`     | DNS lookup timed out                  |
| `ErrNoSuchHost`        | `"Mail server does not exist"`                          | Domain has no DNS entry at all        |
| `ErrServerUnavailable` | `"Mail server is unavailable"`                          | DNS server returned a failure code    |
| `ErrBlocked`           | `"Blocked by mail server"`                              | Request was rejected at network level |

### Handling Pattern

```go
mx, err := verifier.CheckMX(domain)
if err != nil {
    var lookupErr *emailverifier.LookupError
    if errors.As(err, &lookupErr) {
        // Treat as a data signal, not a fatal application error.
        log.Warnf("MX lookup failed for %s: %s (%s)", domain, lookupErr.Message, lookupErr.Details)
        return &emailverifierv1.MXResult{HasMxRecord: false}, nil
    }
    // Unexpected error type — propagate upward.
    return nil, err
}
```

### Guidance

- DNS failures are transient infrastructure events. Log them and return a gracefully degraded `VerifyResponse` rather than a 500.
- Never expose `LookupError.Details` to end users — it may contain internal hostnames or raw DNS server responses. Use `LookupError.Message` for any user-facing messaging.

---

## D-11: Self-Hosted API Server

**Purpose:** The repository ships a ready-to-deploy HTTP API server (`cmd/apiserver`) that wraps `Verify()` and returns JSON. With the adoption of `protojson`, the server's response serialization must use `protojson.Marshal` on `VerifyResponse`.

### Endpoint

```
GET https://{your_host}/v1/{email}/verification
```

The `{email}` path segment is the URL-encoded email address to verify.

### Response

Content-Type: `application/json`. Body is the `protojson`-serialized `VerifyResponse`.

**Example response:**

```json
{
  "email": "user@exampledomain.org",
  "syntax": {
    "username": "user",
    "domain": "exampledomain.org",
    "valid": true
  },
  "mx": {
    "has_mx_record": true,
    "records": [
      { "host": "mail.exampledomain.org", "pref": 10 }
    ]
  },
  "disposable": false,
  "role_account": false,
  "free": false
}
```

### Handler Sketch

```go
func handleVerify(w http.ResponseWriter, r *http.Request) {
    email := chi.URLParam(r, "email") // or equivalent router extraction

    resp, err := VerifyEmail(email) // from D-9
    if err != nil {
        http.Error(w, "verification failed", http.StatusInternalServerError)
        return
    }

    marshaler := protojson.MarshalOptions{
        UseProtoNames:   true,
        EmitUnpopulated: false,
    }
    jsonBytes, err := marshaler.Marshal(resp)
    if err != nil {
        http.Error(w, "serialization failed", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(jsonBytes)
}
```

### Deployment Notes

- The `cmd/apiserver` binary is a reference implementation; add rate limiting, authentication, and observability before exposing externally.
- The `suggestion` field is omitted from the response JSON when empty (via `EmitUnpopulated: false`).

---

## Appendix A: Feature Toggle Reference

| Feature | Default | Opt-in | Opt-out |
|---------|---------|--------|---------|
| Syntax validation | ✅ Always on | — | — |
| MX record check | ✅ Always on | — | — |
| Disposable domain check | ✅ Always on | — | — |
| Free provider check | ✅ Always on | — | — |
| Role account check | ✅ Always on | — | — |
| Domain typo suggestion | ❌ Off | `EnableDomainSuggest()` | `DisableDomainSuggest()` |
| Custom disposable domains | ❌ Off | `AddDisposableDomains([]string)` | — |

---
