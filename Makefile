GO ?= go
GO_BUILD_FLAGS ?=
DOCKER_BUILD_FLAGS ?=
IMAGE_TAG ?= latest

.PHONY: all
all: test

corgi: $(shell find . -iname "*.go") # Build the main binary
	CGO_ENABLED=0 $(GO) build $(GO_BUILD_FLAGS) \
			    -mod=vendor \
			    -o $@ .

.PHONY: test
test: corgi # Build and run the tests
	$(GO) test -mod=vendor ./...

.PHONY: clean
clean: # Clean the local generated artifacts
	rm -fr -- corgi
