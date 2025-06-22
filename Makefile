APP_NAME := node
BIN_DIR := bin
GOPATH := $(shell go env GOPATH)
.PHONY: all build run test fmt vet lint check clean

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
	@go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

run: build
	./$(BIN_DIR)/$(APP_NAME)

test:
	@go test ./... | grep -v '^?' || true

fmt:
	@go fmt ./...
	@$(GOPATH)/bin/goimports -w .

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