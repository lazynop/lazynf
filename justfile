set shell := ["bash", "-c"]

default:
    @just --list

build:
    go build -o bin/vellum .

# Run vellum directly via `go run` (no need to build first).
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
