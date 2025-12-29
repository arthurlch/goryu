GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
GOCLEAN=$(GOCMD) clean
BINARY_NAME=goryu
PKG_LIST := $(shell $(GOCMD) list ./... | grep -v /vendor/)

all: test

test:
	@echo "Running tests..."
	$(GOTEST) -v $(PKG_LIST)

fmt:
	@echo "Formatting code..."
	$(GOFMT) $(PKG_LIST)

vet:
	@echo "Running go vet..."
	$(GOVET) $(PKG_LIST)

clean:
	@echo "Cleaning up..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME) # Remove the built binary
	# Add any other cleanup commands here (e.g., removing coverage files)
	rm -f coverage.out

lint:
	@echo "Running linter (go vet)..."
	$(GOVET) $(PKG_LIST)

install-cli:
	@echo "Installing Goryu CLI..."
	$(GOCMD) install ./cmd/goryu

help:
	@echo "Available targets:"
	@echo "  all        - Test the application (default)"
	@echo "  test       - Run tests"
	@echo "  fmt        - Format Go source code"
	@echo "  vet        - Run go vet"
	@echo "  lint       - Run linter (go vet)"
	@echo "  install-cli - Install Goryu CLI locally"
	@echo "  clean      - Clean up test cache"

.PHONY: all test fmt vet clean lint helps