.PHONY: build build-all test test-race lint vet clean man completions release fmt tidy

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/pmtop/pmtop/internal/version.Version=$(VERSION) \
	-X github.com/pmtop/pmtop/internal/version.Commit=$(COMMIT) \
	-X github.com/pmtop/pmtop/internal/version.Date=$(DATE)

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o build/pmtop ./cmd/pmtop

build-all:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o build/pmtop-linux-amd64 ./cmd/pmtop
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o build/pmtop-linux-arm64 ./cmd/pmtop

test:
	go test -race -coverprofile=coverage.out ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || go vet ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

man:
	go run ./cmd/pmtop man --output-dir man

completions:
	go run ./cmd/pmtop completion bash > completions/bash/pmtop
	go run ./cmd/pmtop completion zsh  > completions/zsh/_pmtop
	go run ./cmd/pmtop completion fish > completions/fish/pmtop.fish

release:
	goreleaser release --clean

clean:
	rm -rf build/ dist/ coverage.out coverage.html
