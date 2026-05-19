GO ?= go
GOFMT ?= gofmt
GOFILES := $(shell find cmd internal -name '*.go' -type f)
DIST_DIR ?= dist

.PHONY: build test fmt vet check release clean

build:
	$(GO) build ./...

test:
	$(GO) test ./...

fmt:
	$(GOFMT) -w $(GOFILES)

vet:
	$(GO) vet ./...

check:
	@unformatted="$$( $(GOFMT) -l $(GOFILES) )"; \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted files:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	$(GO) vet ./...
	$(GO) test ./...
	$(GO) build ./...

release:
	sh scripts/release.sh

clean:
	rm -rf $(DIST_DIR)
