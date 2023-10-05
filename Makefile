.PHONY: test test-examples docs fmt vet

test: ## Run unit tests
	go test ./... -v -covermode=atomic -coverprofile=coverage.out

test-examples: ## Run unit tests for examples directory
	cd examples && go test -v ./... && \
	cd sequence-diagrams-with-sqlite-database && make test && cd ..

.DEFAULT_GOAL := help
help:  ## Show this help
	@grep -E '^[0-9a-zA-Z_-]+[[:blank:]]*:.*?## .*$$' $(MAKEFILE_LIST) | sort \
	| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[1;32m%-15s\033[0m %s\n", $$1, $$2}'