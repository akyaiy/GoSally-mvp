APP_NAME := node
BIN_DIR := bin
GOPATH := $(shell go env GOPATH)
export CONFIG_PATH := ./config.yaml
export NODE_PATH := $(shell pwd)

NODE_VERSION := v0.0.1-dev
SV1_VERSION := v0.0.1-dev

LDFLAGS := -X 'github.com/akyaiy/GoSally-mvp/src/internal/engine/config.NodeVersion=$(NODE_VERSION)' -X 'github.com/akyaiy/GoSally-mvp/src/internal/server/sv1.SV1Version=$(SV1_VERSION)'
CGO_CFLAGS := -I/usr/local/include
CGO_LDFLAGS := -L/usr/local/lib -llua5.1 -lm -ldl
.PHONY: all build run runq test fmt vet lint check clean

all: build

lint-setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

goimports-setup:
	go install golang.org/x/tools/cmd/goimports@latest

golicenses-setup:
	go install github.com/google/go-licenses@latest

setup: lint-setup goimports-setup golicenses-setup
	@echo "Setting up the development environment..."
	@mkdir -p $(BIN_DIR)
	@echo "Setup complete. Run 'make build' to compile the application."

build:
	@echo "Building..."
	@# @echo "CGO_CFLAGS is: '$(CGO_CFLAGS)'"
	@# @echo "CGO_LDFLAGS is: '$(CGO_LDFLAGS)'"
	@# CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)"
	cd src && go build -trimpath -ldflags "-w -s $(LDFLAGS)" -o ../$(BIN_DIR)/$(APP_NAME) ./
# 	@if ! command -v upx >/dev/null 2>&1; then \
# 		echo "upx not found, skipping compression."; \
# 	elif upx -t $(BIN_DIR)/$(APP_NAME) >/dev/null 2>&1; then \
# 		echo "$(BIN_DIR)/$(APP_NAME) already compressed, skipping."; \
# 	else \
# 		upx $(BIN_DIR)/$(APP_NAME) >/dev/null 2>&1 || true; \
# 	fi

run: build
	@echo "Running!"
	exec ./$(BIN_DIR)/$(APP_NAME)

runq: build
	@echo "Running!"
	exec ./$(BIN_DIR)/$(APP_NAME) | jq

pure-run:
	@echo "Running!"
	exec ./$(BIN_DIR)/$(APP_NAME)

test:
	@go test ./... | grep -v '^?' || true

fmt:
	@go fmt ./internal/./...
	@go fmt ./cmd/./...
	@go fmt ./hooks/./...
	@$(GOPATH)/bin/goimports -w ./internal/
	@$(GOPATH)/bin/goimports -w ./cmd/
	@$(GOPATH)/bin/goimports -w ./hooks/

vet:
	@go vet ./...

lint:
	@$(GOPATH)/bin/golangci-lint run

check: fmt vet lint test

licenses:
	@$(GOPATH)/bin/go-licenses save ./... --save_path=third_party/licenses --force
	@echo "Licenses have been exported to third_party/licenses"

clean:
	@rm -rf bin

help:
	@echo "Available commands: $$(cat Makefile | grep -E '^[a-zA-Z_-]+:.*?' | grep -v -- '-setup:' | sed 's/:.*//g' | sort | uniq | tr '\n' ' ')"
