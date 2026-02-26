# ================================================
# Goclaw Makefile
# ================================================

# Variables
APP_NAME := goclaw
MODULE := github.com/goclaw/goclaw
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go settings
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOVET := $(GOCMD) vet

# Build flags
LDFLAGS := -ldflags "\
	-X '$(MODULE)/pkg/version.Version=$(VERSION)' \
	-X '$(MODULE)/pkg/version.BuildTime=$(BUILD_TIME)' \
	-X '$(MODULE)/pkg/version.GitCommit=$(COMMIT_HASH)' \
"
BUILD_FLAGS := -v $(LDFLAGS)

# Directories
BIN_DIR := ./bin
CMD_DIR := ./cmd
PKG_DIR := ./pkg
INTERNAL_DIR := ./internal

# Source files
SRC_FILES := $(shell git ls-files '*.go' 2>/dev/null)
ifeq ($(SRC_FILES),)
SRC_FILES := $(shell find . -type f -name '*.go' -not -path './vendor/*' -not -path './.git/*')
endif

# ================================================
# Default target
# ================================================
.PHONY: all
all: clean fmt vet test build

# ================================================
# Build targets
# ================================================
.PHONY: build-ui
build-ui: ## Build Web UI assets
	@echo "🌐 Building Web UI..."
	@if [ ! -f web/package.json ]; then \
		echo "ℹ️  web/package.json not found, skipping UI build"; \
		exit 0; \
	fi
	@cd web && npm install && npm run build
	@mkdir -p pkg/api/web
	@rm -rf pkg/api/web/dist
	@cp -R web/dist pkg/api/web/
	@echo "✅ Web UI build complete: web/dist"

.PHONY: build
build: ## Build the application binary
	@echo "🔨 Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	@BUILD_TAGS=""; \
	if [ -f web/package.json ]; then \
		$(MAKE) build-ui; \
		BUILD_TAGS="-tags embed_ui"; \
	fi; \
	$(GOBUILD) $$BUILD_TAGS $(BUILD_FLAGS) -o $(BIN_DIR)/$(APP_NAME) $(CMD_DIR)/$(APP_NAME)/main.go
	@echo "✅ Build complete: $(BIN_DIR)/$(APP_NAME)"

.PHONY: build-all
build-all: ## Build for all platforms (linux, darwin, windows)
	@echo "🔨 Building for all platforms..."
	@mkdir -p $(BIN_DIR)
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-linux-amd64 $(CMD_DIR)/$(APP_NAME)/main.go
	# Darwin AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-darwin-amd64 $(CMD_DIR)/$(APP_NAME)/main.go
	# Darwin ARM64 (M1/M2)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-darwin-arm64 $(CMD_DIR)/$(APP_NAME)/main.go
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-windows-amd64.exe $(CMD_DIR)/$(APP_NAME)/main.go
	@echo "✅ Cross-platform build complete"

.PHONY: build-race
build-race: ## Build with race detector enabled
	@echo "🔨 Building with race detector..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -race $(BUILD_FLAGS) -o $(BIN_DIR)/$(APP_NAME)-race $(CMD_DIR)/$(APP_NAME)/main.go
	@echo "✅ Build complete (with race detector)"

# ================================================
# Run targets
# ================================================
.PHONY: run
run: build ## Build and run the application
	@echo "🚀 Running $(APP_NAME)..."
	$(BIN_DIR)/$(APP_NAME)

.PHONY: run-dev
run-dev: ## Run with hot reload (requires air)
	@echo "🔄 Running with hot reload..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "⚠️  air is not installed. Install with: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi

# ================================================
# Test targets
# ================================================
.PHONY: test
test: ## Run all tests
	@echo "🧪 Running tests..."
	$(GOTEST) -v -race ./...

.PHONY: test-short
test-short: ## Run short tests only
	@echo "🧪 Running short tests..."
	$(GOTEST) -v -short ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "🧪 Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

.PHONY: test-coverage-xml
test-coverage-xml: ## Run tests with coverage XML report (requires gocov and gocov-xml)
	@echo "🧪 Running tests with XML coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -func=coverage.out
	@if command -v gocov > /dev/null && command -v gocov-xml > /dev/null; then \
		gocov convert coverage.out | gocov-xml > coverage.xml; \
		echo "✅ XML coverage report generated: coverage.xml"; \
	else \
		echo "⚠️  gocov or gocov-xml not installed. Install with: go install github.com/axw/gocov/gocov@latest && go install github.com/AlekSi/gocov-xml@latest"; \
	fi

.PHONY: test-benchmark
test-benchmark: ## Run benchmark tests
	@echo "⚡ Running benchmark tests..."
	$(GOTEST) -bench=. -benchmem ./...

# ================================================
# Code quality targets
# ================================================
.PHONY: fmt
fmt: ## Format Go code
	@echo "📝 Formatting code..."
	$(GOFMT) -w -s $(SRC_FILES)
	@echo "✅ Code formatted"

.PHONY: vet
vet: ## Run go vet
	@echo "🔍 Running go vet..."
	$(GOVET) ./...

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint to be installed)
	@echo "🔍 Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "⚠️  golangci-lint is not installed. Install from: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

.PHONY: check
check: fmt vet test lint ## Run all code quality checks
	@echo "✅ All checks passed!"

# ================================================
# Dependency management targets
# ================================================
.PHONY: deps
deps: ## Download and verify dependencies
	@echo "📦 Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify

.PHONY: deps-tidy
deps-tidy: ## Tidy and verify go.mod
	@echo "🧹 Tidying dependencies..."
	$(GOMOD) tidy
	$(GOMOD) verify

.PHONY: deps-update
deps-update: ## Update all dependencies
	@echo "⬆️  Updating dependencies..."
	$(GOCMD) get -u ./...
	$(GOMOD) tidy

.PHONY: deps-list
deps-list: ## List all direct dependencies
	@echo "📋 Direct dependencies:"
	$(GOCMD) list -m -f '{{.Path}} {{.Version}}' all | grep -v indirect

# ================================================
# Clean targets
# ================================================
.PHONY: clean
clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BIN_DIR)
	@rm -rf web/dist web/node_modules
	@rm -rf pkg/api/web/dist
	@rm -f coverage.out coverage.html coverage.xml
	@echo "✅ Clean complete"

# ================================================
# Development targets
# ================================================
.PHONY: generate
generate: ## Run go generate
	@echo "🔄 Running go generate..."
	$(GOCMD) generate ./...

.PHONY: proto
proto: ## Generate protobuf files (requires protoc and protoc-gen-go)
	@echo "📄 Generating protobuf files..."
	@if command -v protoc > /dev/null; then \
		mkdir -p pkg/grpc/pb/v1; \
		protoc --go_out=. --go_opt=module=$(MODULE) \
			--go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
			--proto_path=api/proto \
			api/proto/goclaw/v1/*.proto; \
		echo "✅ Protobuf files generated"; \
	else \
		echo "⚠️  protoc is not installed. Please install Protocol Buffers compiler"; \
		exit 1; \
	fi

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "🐳 Running Docker container..."
	docker run --rm -it $(APP_NAME):latest

# ================================================
# Help target
# ================================================
.PHONY: help
help: ## Display this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

# ================================================
# Version info
# ================================================
.PHONY: version
version: ## Display version information
	@echo "App Name:    $(APP_NAME)"
	@echo "Version:     $(VERSION)"
	@echo "Build Time:  $(BUILD_TIME)"
	@echo "Git Commit:  $(COMMIT_HASH)"
	@echo "Go Version:  $(shell go version)"
