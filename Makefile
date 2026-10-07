GO ?= go
GOLANGCI_LINT ?= golangci-lint
BIN_DIR ?= bin

.DEFAULT_GOAL := all

.PHONY: all build test test-cover vet lint fmt tidy verify check clean

all: check build

build:
	$(GO) build -trimpath -o "$(BIN_DIR)/" ./cmd/examsim

test:
	$(GO) test -count=1 ./...

test-cover:
	$(GO) test -count=1 -cover ./...

vet:
	$(GO) vet ./...

lint:
	$(GOLANGCI_LINT) run

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy

verify:
	$(GO) mod verify

check: verify vet lint test

clean:
	$(GO) clean ./...
	$(RM) -r "$(BIN_DIR)"
