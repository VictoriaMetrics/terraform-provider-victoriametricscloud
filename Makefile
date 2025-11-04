.PHONY: help build install test testacc lint fmt clean docs

# Default target
.DEFAULT_GOAL := help

# Variables
HOSTNAME=registry.terraform.io
NAMESPACE=VictoriaMetrics
NAME=victoriametricscloud
BINARY=terraform-provider-${NAME}
VERSION=0.1.0
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)
INSTALL_PATH=~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the provider binary
	@echo "Building provider..."
	go build -o ${BINARY}

install: build ## Install provider locally for testing
	@echo "Installing provider to ${INSTALL_PATH}..."
	mkdir -p ${INSTALL_PATH}
	cp ${BINARY} ${INSTALL_PATH}
	@echo "Provider installed successfully!"
	@echo "You can now use it in your Terraform configurations with:"
	@echo ""
	@echo "terraform {"
	@echo "  required_providers {"
	@echo "    victoriametricscloud = {"
	@echo "      source  = \"${HOSTNAME}/${NAMESPACE}/${NAME}\""
	@echo "      version = \"${VERSION}\""
	@echo "    }"
	@echo "  }"
	@echo "}"

test: ## Run unit tests
	@echo "Running unit tests..."
	go test -v -timeout=120s -parallel=4 ./...

testacc: ## Run acceptance tests (requires VMCLOUD_API_KEY env var)
	@echo "Running acceptance tests..."
	TF_ACC=1 go test -v -timeout 120m ./internal/provider/

lint: ## Run linters
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, installing..." && go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest)
	golangci-lint run --timeout=5m

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	terraform fmt -recursive ./examples/ 2>/dev/null || true

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

license: ## Check dependencies licences
	@echo "Running go license check..."
	which wwhrd || go install github.com/frapposelli/wwhrd@latest
	wwhrd check -f .wwhrd.yml

vulnerabilities: ## Check for vulnerabilities
	@echo "Running vulnerabilities check...."
	which govulncheck || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f ${BINARY}
	go clean

docs: ## Generate documentation
	@echo "Generating documentation..."
	@which tfplugindocs > /dev/null || (echo "tfplugindocs not found, installing..." && go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest)
	tfplugindocs generate

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	go mod vendor

tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	go install github.com/frapposelli/wwhrd@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

dev-override: build ## Setup local provider override for development
	@echo "Setting up local provider override..."
	@mkdir -p ~/.terraformrc.d
	@echo 'provider_installation {' > ~/.terraformrc.d/override.tfrc
	@echo '  dev_overrides {' >> ~/.terraformrc.d/override.tfrc
	@echo '    "${HOSTNAME}/${NAMESPACE}/${NAME}" = "'$(shell pwd)'"' >> ~/.terraformrc.d/override.tfrc
	@echo '  }' >> ~/.terraformrc.d/override.tfrc
	@echo '  direct {}' >> ~/.terraformrc.d/override.tfrc
	@echo '}' >> ~/.terraformrc.d/override.tfrc
	@echo "Override configuration created at ~/.terraformrc.d/override.tfrc"
	@echo "Export: export TF_CLI_CONFIG_FILE=~/.terraformrc.d/override.tfrc"
	@echo "Then run Terraform commands from examples directory"

dev-override-clean: ## Remove local provider override
	@echo "Removing local provider override..."
	@rm -f ~/.terraformrc.d/override.tfrc
	@echo "Override configuration removed"

verify: fmt vet lint test license vulnerabilities ## Run all verification steps

all: verify build ## Run all targets
