# conform Makefile — strict QA gates, fleet pattern (reference: ~/Projects/ferret).
#
# Primary: check (vet+lint+test+build) | audit (check+race+vuln+dupe+nilcheck)

.DEFAULT_GOAL := check

SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c

# ── Shared sandbox (go-sandbox) ──
# doctor ← Makefile.doctor.mk; cross / cross-amd64 / cross-arm64 ← Makefile.cross.mk
include .sandbox/lib/Makefile.doctor.mk
include .sandbox/lib/Makefile.cross.mk

.PHONY: help check audit vet lint test race build vuln dupe nilcheck install deploy clean

# Serialize golangci-lint through the machine-global mkdir mutex (see script
# header — golangci-lint's cache lock fails exit-3 on contention instead of
# waiting, which cascades across parallel sessions/worktrees).
GOLANGCILINT := bash scripts/lint-locked

help: ## Show this help
	@printf '\n\033[1mFour verbs. Identical in every dkoosis repo.\033[0m\n\n'
	@printf '  \033[36mcheck \033[0m  fast gate — vet + lint + test + build. Pre-commit; required in CI.\n'
	@printf '  \033[36maudit \033[0m  check, plus race + vuln + dupe + nilcheck. Before you ask for review.\n'
	@printf '  \033[36mdeploy\033[0m  build + install this tool locally.\n'
	@printf '  \033[36mhelp  \033[0m  this text.\n\n'
	@printf 'Everything below is an internal step of one of those four. Call the verbs.\n\n'
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_.-]+:.*?## / { printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@printf '\n'

check: vet lint test build ## Fast validation: vet + lint + test + build
	@echo "=== check pass ==="

audit: check race vuln dupe nilcheck ## Exhaustive validation
	@echo "=== audit pass ==="

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint (strict config)
	$(GOLANGCILINT) run ./...

# No -count=1: Go's test cache stays ON for the dev loop; race/audit bypass it.
test: ## Run tests with coverage
	go test -cover ./...

race: ## Run tests with race detector (fresh run)
	go test -race -count=1 -cover ./...

build: ## Compile everything
	go build ./...

vuln: ## Scan for known vulnerabilities
	govulncheck ./...

# jscpd's --gitignore flag no longer exists (silent no-op in older Makefiles);
# --ignore with explicit globs is the working form.
dupe: ## Check for code duplication (jscpd)
	TMP_JSCPD=$$(mktemp -d); jscpd . --ignore "**/.git/**,**/.sandbox/**,**/.beads/**" --output $$TMP_JSCPD; rm -rf $$TMP_JSCPD

nilcheck: ## Run nilaway (skips if not installed)
	@if ! command -v nilaway >/dev/null 2>&1; then \
		echo "nilcheck: nilaway not installed — skipping (install: go install go.uber.org/nilaway/cmd/nilaway@latest)"; \
		exit 0; \
	fi
	nilaway -include-pkgs=github.com/dkoosis/conform ./...

## doctor target provided by .sandbox/lib/Makefile.doctor.mk (project.conf-driven)
## cross / cross-amd64 / cross-arm64 provided by .sandbox/lib/Makefile.cross.mk

install: ## Install conform to GOPATH/bin
	go install ./cmd/conform

deploy: build install ## Build, then install conform to GOPATH/bin
	@echo "=== deployed (conform installed to $$(go env GOPATH)/bin) ==="

clean: ## Remove built binary + sandbox build artifacts
	@rm -f conform
	@rm -rf .sandbox/bin/linux-amd64 .sandbox/bin/linux-arm64 .sandbox/cache
	@echo "=== clean ==="
