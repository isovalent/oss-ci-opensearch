GO ?= go
GO_BUILD_FLAGS ?=
DOCKER_BUILD_FLAGS ?=
IMAGE_TAG ?= latest

.PHONY: all
all: test

corgi: $(shell find . -iname "*.go")
	CGO_ENABLED=0 $(GO) build $(GO_BUILD_FLAGS) \
			    -mod=vendor \
			    -o $@ .

.PHONY: test
test: corgi
	$(GO) test -mod=vendor ./...

.PHONY: clean
clean:
	rm -fr -- corgi
