SOURCE_DIRS      = handler leader predicate status
SOURCES          := $(shell find . -name '*.go' -not -path "*/vendor/*" -not -path "*/.git/*")
.DEFAULT_GOAL    := build

# ensure: ## Install or update project dependencies
#     @dep ensure

build: $(SOURCES) ## Build Test
	go build -i -ldflags="-s -w" ./...

lint: golangci-lint ## Run golint
	@$(GOLANGCI_LINT) run

fmt: ## Run go fmt
	@gofmt -d $(SOURCES)

fmtcheck: ## Check go formatting
	@gofmt -l $(SOURCES) | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi

test: ## Run unit tests
	@go test -race -covermode atomic -coverprofile cover.out $(addprefix ./, $(addsuffix /... , $(SOURCE_DIRS)))

vet: ## Run go vet
	@go vet $(addprefix ./, $(SOURCE_DIRS))

tidy: ## Tidy go dependencies
	@go mod tidy

check-license: $(SOURCES)
	@./hack/check-license.sh "$(SOURCES)"

check: tidy fmtcheck vet lint build test check-license ## Pre-flight checks before creating PR
	@git diff --exit-code

clean: ## Clean up your working environment
	@rm -f coverage-all.out coverage.out

golangci-lint:
ifeq (, $(shell which golangci-lint))
	@{ \
	set -e ;\
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.27.0 ;\
	}
GOLANGCI_LINT=$(shell go env GOPATH)/bin/golangci-lint
else
GOLANGCI_LINT=$(shell which golangci-lint)
endif

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

.PHONY: ensure build lint fmt fmtcheck test vet check help clean
