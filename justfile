set shell := ["bash", "-c"]

# Embedded into the binary via -ldflags. Prefers `git describe` (e.g. v0.1.0 -> 0.1.0),
# falls back to the short commit hash, and finally to "0.0.1-dev" outside a git checkout.
VERSION := `git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' | grep -v '^$' || echo "0.0.1-dev"`

default:
    @just --list

build:
    go build -ldflags "-X main.version={{VERSION}}" -o bin/lazynf .

# Run lazynf directly via `go run` (no need to build first).
# Pass any arguments after `just run`, e.g.:
#   just run --help
#   just run list
#   just run search mono
#   just run install JetBrainsMono
#   just run -v install FiraCode
run *ARGS:
    @go run . {{ARGS}}

test:
    go test ./... -v

fmt:
    gofmt -w .

fmt-check:
    @diff -u <(echo -n) <(gofmt -d .)

vet:
    go vet ./...

check: fmt-check vet test
