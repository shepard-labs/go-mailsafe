.PHONY: all build test vet lint clean run-server fmt

all: build test

# Build all packages.
build:
	go build ./...

# Run the full test suite.
test:
	go test -v -count=1 ./...

# Run go vet.
vet:
	go vet ./...

# Format all Go files.
fmt:
	gofmt -w .

# Remove build artifacts.
clean:
	rm -f mailsafe-server

# Run the API server locally.
run-server:
	go run ./cmd/apiserver