APP := containsentry
GO ?= go

.PHONY: help build test fmt check

help:
	@echo "Available targets:"
	@echo "  make build"
	@echo "  make test"
	@echo "  make fmt"
	@echo "  make check"

build:
	$(GO) build -o $(APP) ./cmd/containsentry

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

check: test build
