SOURCES          := $(shell find . -name '*.go' -not -path "*/vendor/*" -not -path "*/.git/*")
.DEFAULT_GOAL    := build

# bingo manages consistent tooling versions for things like kind, kustomize, etc.
include .bingo/Variables.mk

build: $(SOURCES) ## Build Test
	go build -ldflags="-s -w" ./...

lint: golangci-lint ## Run golangci-lint
	@$(GOLANGCI_LINT) run

lint-fix: golangci-lint ## Run golangci lint to automatically perform fixes
	@$(GOLANGCI_LINT) run --fix

fmt: ## Run go fmt
	@go fmt ./...

fmtcheck: ## Check go formatting
	@gofmt -l $(SOURCES) | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi

test: ## Run unit tests
	@go test -race -covermode atomic -coverprofile cover.out ./...

vet: ## Run go vet
	@go vet ./...

tidy: ## Tidy go dependencies
	@go mod tidy

check-license: $(SOURCES) ## Check license headers
	@./hack/check-license.sh "$(SOURCES)"

check: tidy fmtcheck vet lint build test check-license ## Pre-flight checks before creating PR
	@git diff --exit-code

clean: ## Clean up your working environment
	@rm -f cover.out

# generate: ## regenerate mocks
#     go get github.com/vektra/mockery/.../
#     @go generate ./...

help: ## Show this help screen
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build lint lint-fix fmt fmtcheck test vet tidy check-license check clean golangci-lint help
