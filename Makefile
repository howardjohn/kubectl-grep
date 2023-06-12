.ONESHELL:
SHELL := /bin/bash
GOBIN ?= $(GOPATH)/bin
MODULE = github.com/howardjohn/kubectl-grep
export GO111MODULE ?= on

all: format lint install

.PHONY: deps
deps: $(GOBIN)/goimports $(GOBIN)/golangci-lint

.PHONY: check-git
check-git:
	@
	if [[ -n $$(git status --porcelain) ]]; then
		echo "Error: git is not clean"
		git status
		git diff
		exit 1
	fi

.PHONY: gen-check
gen-check: format check-git

.PHONY: format
format:
	@go mod tidy
	@gofumpt -w .
	@goimports -local $(MODULE) -w .
	@gci write -s standard -s default -s Prefix\($(MODULE)\) .

.PHONY: lint
lint:
	@golangci-lint run --fix

.PHONY: install
install:
	@go install

.PHONY: test
test:
	@go test ./...
