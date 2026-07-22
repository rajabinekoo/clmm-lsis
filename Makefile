APP_NAME := clmm-lsis
MODULE := github.com/rajabinekoo/clmm-lsis

GO ?= go
CONFIG ?= configs/study.example.json

BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)

VERSION ?= 0.1.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make fmt           Format all Go files"
	@echo "  make test          Run all unit tests"
	@echo "  make vet           Run go vet"
	@echo "  make check         Run fmt-check, vet, and tests"
	@echo "  make build         Build the CLI"
	@echo "  make config-check  Validate the study configuration"
	@echo "  make postgres-up   Start PostgreSQL"
	@echo "  make postgres-down Stop PostgreSQL"
	@echo "  make clean         Remove build artifacts"

.PHONY: fmt
fmt:
	@$(GO) fmt ./...

.PHONY: fmt-check
fmt-check:
	@test -z "$$($(GO) fmt ./...)" || \
		(echo "Go files were not formatted. Run 'make fmt'." && exit 1)

.PHONY: test
test:
	@$(GO) test ./...

.PHONY: vet
vet:
	@$(GO) vet ./...

.PHONY: check
check: fmt-check vet test

.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	@$(GO) build \
		-trimpath \
		-ldflags "$(LDFLAGS)" \
		-o $(BIN_PATH) \
		./cmd/clmm-lsis

.PHONY: config-check
config-check:
	@$(GO) run ./cmd/clmm-lsis config-check --config $(CONFIG)

.PHONY: version
version:
	@$(GO) run ./cmd/clmm-lsis version

.PHONY: postgres-up
postgres-up:
	@docker compose up -d postgres

.PHONY: postgres-down
postgres-down:
	@docker compose down

.PHONY: clean
clean:
	@rm -rf $(BIN_DIR)