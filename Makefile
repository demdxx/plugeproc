.PHONY: all
all: lint cover


.PHONY: lint
lint: golint ## Run linter


.PHONY: golint
golint:
	golangci-lint run -v ./...


.PHONY: fmt
fmt: ## Run formatting code
	@echo "Fix formatting"
	@gofmt -w ${GO_FMT_FLAGS} $$(go list -f "{{ .Dir }}" ./...); if [ "$${errors}" != "" ]; then echo "$${errors}"; fi


.PHONY: test
test: ## Run all package tests
	go test -race -v ./...


.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy


.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
