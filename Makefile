GO ?= go
GO_BUILD_FLAGS ?=
DOCKER_BUILD_FLAGS ?=
IMAGE_TAG ?= latest

.PHONY: all
all: test

opensearch-ingester: $(shell find . -iname "*.go")
	CGO_ENABLED=0 $(GO) build $(GO_BUILD_FLAGS) \
			    -mod=vendor \
			    -o $@ .

.PHONY: test
test: opensearch-ingester
	$(GO) test -mod=vendor ./...

.PHONY: clean
clean:
	rm -fr opensearch-ingester
