set shell := ["bash", "-c"]

default:
    @just --list

build:
    go build -o bin/vellum .

test:
    go test ./... -v

fmt:
    gofmt -w .

fmt-check:
    @diff -u <(echo -n) <(gofmt -d .)

vet:
    go vet ./...

check: fmt-check vet test
