BINARY      ?= dist/todoList
MAIN        := todoList.go
GO          ?= go
UPX_BIN     ?= upx
UPX_FLAGS   ?= --lzma

PORT        ?= 8080
DB_NAME     ?= todoList.db

IMAGE       ?= todoList
IMAGE_TAG   ?= latest
DOCKER      ?= docker

GOOS        ?=
GOARCH      ?=
GOARM       ?=
CC          ?=

CGO_ENABLED ?= 1
CGO_CFLAGS  ?= -Os

STATIC      ?= 0

GOFLAGS     := -trimpath
LDFLAGS     := -s -w -buildid=

ifeq ($(STATIC),1)
LDFLAGS += -linkmode external -extldflags -static
endif

export GOOS GOARCH GOARM CC CGO_ENABLED CGO_CFLAGS

.DEFAULT_GOAL := build
.PHONY: build build-debug pack run clean fmt vet tidy check help docker docker-run

build:
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY) $(MAIN)

build-debug:
	$(GO) build -o $(BINARY) $(MAIN)

pack: build
	@command -v $(UPX_BIN) >/dev/null 2>&1 || { echo "upx not found"; exit 1; }
	env -u UPX $(UPX_BIN) $(UPX_FLAGS) $(BINARY)
	@ls -la $(BINARY)

run: build
	PORT=$(PORT) DB_NAME=$(DB_NAME) ./$(BINARY)

docker:
	$(DOCKER) build -t $(IMAGE):$(IMAGE_TAG) .

docker-run: docker
	$(DOCKER) run --rm -p $(PORT):8080 -v $(IMAGE)-data:/app/data $(IMAGE):$(IMAGE_TAG)

clean:
	rm -f $(BINARY)

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

check: fmt vet

help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
