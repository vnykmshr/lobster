.PHONY: build test test-verbose test-coverage lint clean run install help

# Build configuration
BINARY_NAME=lobster
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0-dev")
BUILD_DIR=.
CMD_DIR=./cmd/lobster
INTERNAL_PACKAGES=./internal/...

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

## help: Display this help message
help:
	@echo "Lobster - Build Targets"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@awk '/^## [a-zA-Z_-]+:/ { sub(/^## /, ""); split($$0, parts, ": "); printf "  %-20s %s\n", parts[1], substr($$0, length(parts[1])+3) }' $(MAKEFILE_LIST)

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -race -timeout 30s $(INTERNAL_PACKAGES)

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	$(GOTEST) -v -race -timeout 30s $(INTERNAL_PACKAGES)

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic $(INTERNAL_PACKAGES)
	@echo "Coverage report generated: coverage.out"
	@$(GOCMD) tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

## test-coverage-html: Generate HTML coverage report
test-coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "HTML coverage report: coverage.html"

## lint: Run linters
lint:
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: brew install golangci-lint" && exit 1)
	golangci-lint run --timeout 5m

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@$(GOCMD) fmt ./...
	@echo "Code formatted"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@$(GOCMD) vet ./...

## tidy: Tidy go modules
tidy:
	@echo "Tidying modules..."
	@$(GOMOD) tidy
	@echo "Modules tidied"

## clean: Remove build artifacts and test files
clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@rm -f *.json *.html
	@echo "Clean complete"

## install: Install the binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOINSTALL) $(LDFLAGS) $(CMD_DIR)
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

## run: Build and run with example arguments
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME) -help

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@echo "Dependencies downloaded"

## verify: Verify dependencies
verify:
	@echo "Verifying dependencies..."
	@$(GOMOD) verify
	@echo "Dependencies verified"

## ci: Run CI pipeline (fmt, vet, lint, test, build)
ci: fmt vet lint test build
	@echo "CI pipeline complete"

## pre-commit: Run checks before committing (fmt, vet, test)
pre-commit: fmt vet test
	@echo "Pre-commit checks passed"

# Default target
.DEFAULT_GOAL := help
