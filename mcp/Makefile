.DEFAULT_GOAL := help

.PHONY: help sync lock lock-update lint format format-check typecheck test test-docker check run

help:
	@printf '%s\n' \
		'Pomelo Orbit MCP commands:' \
		'  make sync          Install locked dependencies' \
		'  make lock          Verify uv.lock is current' \
		'  make lock-update   Resolve and update uv.lock' \
		'  make lint          Run Ruff checks' \
		'  make format        Format Python files with Ruff' \
		'  make format-check  Verify Ruff formatting' \
		'  make typecheck     Run mypy on MCP source' \
		'  make test          Run default pytest suite' \
		'  make test-docker   Run explicit Docker integration test' \
		'  make check         Run lock, lint, formatting, typing, and tests' \
		'  make run           Start the stdio MCP server'

sync:
	uv sync --all-groups

lock:
	uv lock --check

lock-update:
	uv lock

lint:
	uv run ruff check src tests

format:
	uv run ruff format src tests

format-check:
	uv run ruff format --check src tests

typecheck:
	uv run mypy

test:
	uv run pytest

test-docker:
	@test "$$POMELO_ORBIT_RUN_DOCKER_TESTS" = "1" || (printf '%s\n' 'Set POMELO_ORBIT_RUN_DOCKER_TESTS=1.' && exit 2)
	@test -n "$$POMELO_ORBIT_DOCKER_TEST_WORKSPACE" || (printf '%s\n' 'Set POMELO_ORBIT_DOCKER_TEST_WORKSPACE.' && exit 2)
	@test -n "$$POMELO_ORBIT_DOCKER_TEST_PROJECT" || (printf '%s\n' 'Set POMELO_ORBIT_DOCKER_TEST_PROJECT.' && exit 2)
	uv run pytest -m docker

check: lock lint format-check typecheck test

run:
	uv run pomelo-orbit-mcp
