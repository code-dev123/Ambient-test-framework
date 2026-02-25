# Makefile for ambient-test-framework
# ──────────────────────────────────────────────────────────────────────────────

BINARY      := ambient-test
MODULE      := github.com/yourorg/ambient-test-framework
BUILD_DIR   := bin

# Test flags — override on the command line
LAYER       ?= all
KUBECONFIG  ?=
EKS_CLUSTER ?=
REGION      ?= us-west-2
TIMEOUT     ?= 30m
KEEP_NS     ?= false
RUN         ?=

# ──────────────────────────────────────────────────────────────────────────────
# Build targets
# ──────────────────────────────────────────────────────────────────────────────

.PHONY: build
build: ## Build the CLI binary
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/ambient-test/...

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)

# ──────────────────────────────────────────────────────────────────────────────
# Test targets
# ──────────────────────────────────────────────────────────────────────────────

# Build base go test flags from make vars
_TEST_FLAGS := -layer $(LAYER) -timeout $(TIMEOUT)
ifneq ($(KUBECONFIG),)
  _TEST_FLAGS += -kubeconfig $(KUBECONFIG)
endif
ifneq ($(EKS_CLUSTER),)
  _TEST_FLAGS += -eks-cluster $(EKS_CLUSTER) -region $(REGION)
endif
ifeq ($(KEEP_NS),true)
  _TEST_FLAGS += -keep-ns
endif
ifneq ($(RUN),)
  _TEST_FLAGS += -run $(RUN)
endif

.PHONY: test
test: ## Run all tests (LAYER=all by default)
	go test ./test/... -v $(_TEST_FLAGS)

.PHONY: test-l4
test-l4: ## Run only L4 tests
	$(MAKE) test LAYER=l4

.PHONY: test-l7
test-l7: ## Run only L7 tests
	$(MAKE) test LAYER=l7

.PHONY: test-canary
test-canary: ## Run only canary tests
	go test ./test/l7/canary/... -v $(_TEST_FLAGS)

.PHONY: test-baseline
test-baseline: ## Run baseline (no-mesh) tests
	go test ./test/baseline/... -v $(_TEST_FLAGS)

.PHONY: test-unit
test-unit: ## Run unit tests (pkg/ only — no cluster required)
	go test ./pkg/... -v

# ──────────────────────────────────────────────────────────────────────────────
# Code quality
# ──────────────────────────────────────────────────────────────────────────────

.PHONY: fmt
fmt: ## Format all Go source
	gofmt -l -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (install separately)
	golangci-lint run ./...

.PHONY: tidy
tidy: ## Tidy go modules
	go mod tidy

.PHONY: check
check: fmt vet tidy ## Run all code quality checks

# ──────────────────────────────────────────────────────────────────────────────
# Help
# ──────────────────────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
