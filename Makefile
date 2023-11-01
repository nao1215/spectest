.PHONY: test test-examples docs fmt vet

APP         = spectest
VERSION     = $(shell git describe --tags --abbrev=0)
GIT_REVISION := $(shell git rev-parse HEAD)
GO          = go
GO_BUILD    = $(GO) build
GO_TEST     = $(GO) test -v
GO_TOOL     = $(GO) tool
GOOS        = ""
GOARCH      = ""
GO_PKGROOT  = ./...
GO_PACKAGES = $(shell $(GO_LIST) $(GO_PKGROOT))
GO_LDFLAGS  = -ldflags '-X github.com/go-spectest/spectest/version.Version=${VERSION}' -ldflags "-X github.com/go-spectest/spectest/version.Revision=$(GIT_REVISION)"

build:  ## Build binary
	env GO111MODULE=on GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO_BUILD) $(GO_LDFLAGS) -o $(APP) cmd/spectest/main.go

test: ## Run unit tests
	go test ./... -v -covermode=atomic -cover -coverpkg=./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test-examples: ## Run unit tests for examples directory
	make -C examples test

clean: ## Clean up
	rm -f coverage.out coverage.html $(APP)

.DEFAULT_GOAL := help
help:  ## Show this help
	@grep -E '^[0-9a-zA-Z_-]+[[:blank:]]*:.*?## .*$$' $(MAKEFILE_LIST) | sort \
	| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[1;32m%-15s\033[0m %s\n", $$1, $$2}'