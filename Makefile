.PHONY: integration
integration:
	$(MAKE) -C ./integration test

git-hooks:
	@echo "====================="
	@echo "Setting up git hooks..."
	@echo "====================="
	/bin/sh ./scripts/hooks.sh

environment: git-hooks

IMPORTS_REVISER_VERSION ?= v2.5.1
GOLANGCI_LINT_VERSION ?= v1.46.2

imports:
	@echo "====================="
	@echo "Making imports..."
	@echo "====================="
	GOBIN=$(LOCAL_BIN) go install -mod=mod github.com/incu6us/goimports-reviser/v2@$(IMPORTS_REVISER_VERSION)
	find . -name \*.go -not -path "./vendor/*" -not -path "*/pb/*" -exec ./bin/goimports-reviser -file-path {} -rm-unused -set-alias -format \;

lint:
	@echo "====================="
	@echo "Running linter..."
	@echo "====================="
	GOBIN=$(LOCAL_BIN) go install -mod=mod github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	./bin/golangci-lint run --fast --fix