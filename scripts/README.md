# Pomelo Orbit CLI

`scripts/` is the Python `>=3.12` uv project for Pomelo Orbit operational commands. Its `src/` package exposes the project-local `pomelo-orbit-cli` entry point; no global installation is required.

```bash
uv --directory scripts run pomelo-orbit-cli --help
uv --directory scripts run pomelo-orbit-cli database-transfer --help
```

The CLI locates the Orbit repository, then loads the same configuration layers as the Go service: `configs/config.yaml`, optional `configs/config.<env>.yaml`, `.env` or `.env.<env>`, and process environment overrides. Set `POMELO_ORBIT_APP__ENV` in the process to select a profile.

Run the script quality checks with:

```bash
uv --directory scripts run ruff format --check .
uv --directory scripts run ruff check .
uv --directory scripts run mypy
uv --directory scripts run pytest
```
