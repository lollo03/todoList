BINARY      := dist/todoList
MAIN        := todoList.go
GO          ?= go
UPX_BIN     ?= upx
UPX_FLAGS   ?= --lzma

PORT        ?= 8080
DB_NAME     ?= todoList.db

CGO_ENABLED ?= 1
CGO_CFLAGS  ?= -Os

GOFLAGS     := -trimpath
LDFLAGS     := -s -w -buildid=

.DEFAULT_GOAL := build
.PHONY: build build-debug pack run clean fmt vet tidy check help

build:
	CGO_ENABLED=$(CGO_ENABLED) CGO_CFLAGS="$(CGO_CFLAGS)" $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY) $(MAIN)

build-debug:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINARY) $(MAIN)

pack: build
	@command -v $(UPX_BIN) >/dev/null 2>&1 || { echo "upx not found"; exit 1; }
	env -u UPX $(UPX_BIN) $(UPX_FLAGS) $(BINARY)
	@ls -la $(BINARY)

run: build
	PORT=$(PORT) DB_NAME=$(DB_NAME) ./$(BINARY)

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
