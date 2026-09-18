.DEFAULT_GOAL := help

UV ?= uv
UV_RUN ?= $(UV) run --locked --no-sync
CHECK_FIX := $(filter 1 true yes,$(fix))
RUFF_FORMAT_ARGS := --check
RUFF_CHECK_ARGS :=

ifneq ($(CHECK_FIX),)
RUFF_FORMAT_ARGS :=
RUFF_CHECK_ARGS := --fix
endif

.PHONY: help deps check test run

help: ## Show available targets.
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  deps   Sync locked dependencies and install pomelo-orbit-cli"
	@echo "  check  Run Ruff and Mypy quality checks (fix=1 enables fixes)"
	@echo "  test   Run unit tests"
	@echo "  run    Run pomelo-orbit-cli (default POMELO_ORBIT_APP__ENV=development); pass ARGS='database-transfer --help'"

deps: ## Sync locked development dependencies and install the project.
	$(UV) sync --all-groups --locked

check: ## Run Ruff and Mypy checks without tests; use fix=1 to fix Ruff issues.
	$(UV_RUN) ruff format $(RUFF_FORMAT_ARGS) .
	$(UV_RUN) ruff check $(RUFF_CHECK_ARGS) .
	$(UV_RUN) mypy

test: ## Run unit tests.
	$(UV_RUN) pytest

run: export POMELO_ORBIT_APP__ENV ?= development
run: ## Run pomelo-orbit-cli; default POMELO_ORBIT_APP__ENV=development.
	$(UV_RUN) pomelo-orbit-cli $(ARGS)
