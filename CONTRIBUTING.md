# Contributing to shepard-labs/go-mailsafe

Thank you for your interest in contributing to shepard-labs/go-mailsafe. This document provides guidelines and instructions for contributing.

---

## Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Features](#suggesting-features)
  - [Submitting Code Changes](#submitting-code-changes)
- [Code Style](#code-style)
- [Testing](#testing)
- [Updating Metadata Lists](#updating-metadata-lists)
- [Proto Schema Changes](#proto-schema-changes)
- [Pull Request Process](#pull-request-process)
- [License](#license)

---

## Getting Started

1. Fork the repository on GitHub.
2. Clone your fork locally:
   ```bash
   git clone https://github.com/shepard-labs/go-mailsafe.git
   cd go-mailsafe
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/shepard-labs/go-mailsafe.git
   ```
4. Create a branch for your work:
   ```bash
   git checkout -b my-feature
   ```

---

## Development Setup

### Requirements

- **Go 1.25** or later
- **protoc** (Protocol Buffer compiler) — only needed if modifying `.proto` files
- **protoc-gen-go** — Go code generator for protobuf

### Install dependencies

```bash
go mod download
```

### Build

```bash
go build ./...
```

### Run tests

```bash
go test -v ./...
```

### Run the API server locally

```bash
go run ./cmd/apiserver
```

The server starts on `http://localhost:8080` by default.

---

## How to Contribute

### Reporting Bugs

Open a GitHub issue with:

- A clear, descriptive title.
- Steps to reproduce the problem.
- Expected behavior vs. actual behavior.
- Go version (`go version`) and OS.
- Minimal code example that reproduces the issue, if applicable.

### Suggesting Features

Open a GitHub issue tagged as a feature request with:

- A description of the problem the feature would solve.
- Your proposed solution or API design.
- Any alternatives you have considered.

### Submitting Code Changes

1. Ensure your change addresses an open issue or is discussed beforehand for larger features.
2. Write tests that cover your change.
3. Run the full test suite and confirm all tests pass.
4. Submit a pull request against the `main` branch.

---

## Code Style

- Follow standard Go conventions as enforced by `gofmt` and `go vet`.
- Run `gofmt` on all Go files before committing:
  ```bash
  gofmt -w .
  ```
- Run `go vet` to catch common issues:
  ```bash
  go vet ./...
  ```
- Keep functions focused and small. Each `.go` file in the root package corresponds to a single feature or concern.
- Exported types and functions should have documentation comments.
- Do not add dependencies without discussion. The library intentionally has a minimal dependency footprint.

---

## Testing

All changes must include tests. The test file is `verifier_test.go` in the root package.

### Running tests

```bash
go test -v -count=1 ./...
```

### Writing tests

- Test function names should follow the pattern `TestFeatureName_Scenario` (e.g., `TestParseAddress_Valid`, `TestIsDisposable`).
- Test both positive and negative cases.
- For DNS-dependent tests (MX lookups), be aware that results depend on network access. Tests that make real DNS queries should be clearly documented and may be skipped in CI with build tags if needed.
- Use table-driven tests when testing multiple inputs for the same function:
  ```go
  func TestIsRoleAccount_Cases(t *testing.T) {
      cases := []struct {
          input string
          want  bool
      }{
          {"admin", true},
          {"john", false},
          {"postmaster", true},
      }
      v := NewVerifier()
      for _, tc := range cases {
          if got := v.IsRoleAccount(tc.input); got != tc.want {
              t.Errorf("IsRoleAccount(%q) = %v, want %v", tc.input, got, tc.want)
          }
      }
  }
  ```

---

## Updating Metadata Lists

The library bundles several static datasets as Go source files. If you are updating these lists, follow the conventions below.

### Disposable domains (`metadata_disposable.go`)

The canonical upstream source is [disposable-email-domains](https://github.com/disposable-email-domains/disposable-email-domains).

To regenerate:

1. Fetch the latest blocklist:
   ```bash
   curl -sL "https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/master/disposable_email_blocklist.conf" -o /tmp/disposable_domains.txt
   ```
2. Generate the Go file with the domain list as a `[]string` variable named `disposableDomainList`.
3. Verify the build compiles and tests pass.

### Free providers (`metadata_free.go`)

The canonical upstream source is [free-email-domains](https://github.com/Kikobeats/free-email-domains).

To regenerate:

1. Fetch the latest list:
   ```bash
   curl -sL "https://raw.githubusercontent.com/Kikobeats/free-email-domains/master/domains.json" -o /tmp/free_domains.json
   ```
2. Generate the Go file with the domain list as a `map[string]bool` variable named `freeDomainList`.
3. Verify the build compiles and tests pass.

### Role accounts (`metadata_role.go`)

The role account list is a `map[string]bool` named `roleAccountList`. When adding entries:

- Only add usernames that are widely recognized as functional/role-based across email systems and RFCs.
- Use lowercase.

### Suggestion domains (`metadata_suggestion.go`)

The suggestion domain list is a `[]string` named `suggestDomainList`. When adding entries:

- Only add domains that are commonly misspelled by end users.
- The typo engine uses Levenshtein distance, so domains that are very short (3 characters or fewer) may produce false-positive matches. Use judgment.

---

## Proto Schema Changes

The proto schema is defined in `proto/v1/emailverifier.proto`. If you modify it:

1. **Never change existing field numbers.** Add new fields with new numbers.
2. **Never remove fields.** Mark them as deprecated if no longer needed.
3. Regenerate the Go code:
   ```bash
   protoc --go_out=. --go_opt=paths=source_relative proto/v1/emailverifier.proto
   ```
4. Do not manually edit `proto/v1/emailverifier.pb.go` — it is generated code.
5. Update any conversion helpers in `cmd/apiserver/main.go` if the proto messages change.

---

## Pull Request Process

1. **One concern per PR.** Keep pull requests focused. A bug fix and a new feature should be separate PRs.
2. **Descriptive title and body.** Explain what the change does and why. Reference the relevant issue number if applicable.
3. **All tests must pass.** The full test suite (`go test ./...`) must pass before a PR will be reviewed.
4. **Code must compile cleanly.** `go build ./...` and `go vet ./...` must produce no errors or warnings.
5. **Formatting.** All Go code must be formatted with `gofmt`.
6. **Review.** At least one maintainer review is required before merging.
7. **Squash on merge.** PRs are squash-merged to keep the commit history clean.

---

## License

By contributing to shepard-labs/go-mailsafe, you agree that your contributions will be licensed under the [GNU General Public License v3.0](LICENSE).
