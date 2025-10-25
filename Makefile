.PHONY: build install test clean docker-build docker-push lint lint-fix help

# Binary name
BINARY := packnplay

# Container image
IMAGE := ghcr.io/obra/packnplay-default:latest

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOINSTALL := $(GOCMD) install
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean

help: ## Show this help
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build the binary
	$(GOBUILD) -o $(BINARY) .

install: ## Install the binary to GOPATH/bin
	$(GOINSTALL)

test: ## Run tests
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

lint: ## Run golangci-lint
	golangci-lint run

lint-fix: ## Run golangci-lint with auto-fix
	golangci-lint run --fix

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(BINARY)
	rm -f coverage.out coverage.html

docker-build: ## Build the default container image
	docker build -t $(IMAGE) .devcontainer/

docker-test: docker-build ## Test the container image
	@echo "Testing container image..."
	docker run --rm $(IMAGE) which node npm claude codex gemini gh
	docker run --rm $(IMAGE) node --version
	docker run --rm $(IMAGE) npm --version
	docker run --rm $(IMAGE) gh --version

docker-push: docker-build ## Push the container image to GHCR
	docker push $(IMAGE)

all: clean build test ## Clean, build, and test

.DEFAULT_GOAL := help
