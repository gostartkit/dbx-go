GO ?= go
GOFMT ?= gofmt
GOFILES := $(shell find cmd internal -name '*.go' -type f)

.PHONY: build test fmt vet

build:
	$(GO) build ./...

test:
	$(GO) test ./...

fmt:
	$(GOFMT) -w $(GOFILES)

vet:
	$(GO) vet ./...
