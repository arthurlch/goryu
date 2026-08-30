BINARY := goryu
PKG := github.com/arthurlch/goryu/internal/cli
IMAGE := ghcr.io/arthurlch/goryu

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(PKG).version=$(VERSION) \
	-X $(PKG).commit=$(COMMIT) \
	-X $(PKG).date=$(DATE)

.DEFAULT_GOAL := help

## build: compile the CLI with version metadata
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/goryu

## install: install the CLI into GOBIN
install:
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/goryu

## test: run unit tests with the race detector
test:
	go test -race ./...

## cover: run tests and report coverage
cover:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out | tail -1

## cover-html: open the coverage report in a browser
cover-html: cover
	go tool cover -html=coverage.out

## bench: run benchmarks
bench:
	go test -bench=. -benchmem -run=^$$ ./tests/bench/...

## test-cli: run the CLI end-to-end suite
test-cli:
	./test_cli.sh

## fmt: format the code
fmt:
	gofmt -w .

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint (shows the full backlog; CI blocks only new issues)
lint:
	golangci-lint run ./...

## vulncheck: scan for known vulnerabilities
vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

## tidy: sync go.mod and go.sum
tidy:
	go mod tidy

## docker: build the container image locally
docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(DATE) \
		-t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

## snapshot: build a local release snapshot with GoReleaser
snapshot:
	goreleaser release --snapshot --clean

## clean: remove build and coverage artifacts
clean:
	go clean
	rm -f $(BINARY) coverage.out
	rm -rf dist

## help: list available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | awk -F': ' '{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build install test cover cover-html bench test-cli fmt vet lint vulncheck tidy docker snapshot clean help
