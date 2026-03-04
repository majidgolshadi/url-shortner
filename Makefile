.PHONY: build test lint run clean

# Build variables
BINARY_NAME=url-shortener
COMMIT_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TAG=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.CommitHash=$(COMMIT_HASH) -X main.Tag=$(TAG)"

## build: Build the application binary
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/server

## run: Run the application
run: build
	./bin/$(BINARY_NAME)

## test: Run all tests
test:
	go test -race -count=1 ./...

## test-verbose: Run all tests with verbose output
test-verbose:
	go test -race -count=1 -v ./...

## lint: Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

## fmt: Format Go source files
fmt:
	gofmt -s -w .
	goimports -w .

## vet: Run go vet
vet:
	go vet ./...

## clean: Remove build artifacts
clean:
	rm -rf bin/

## help: Show this help message
help:
	@echo "Usage:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'