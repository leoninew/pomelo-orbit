# Pomelo Orbit CLI

`scripts/` is the Python `>=3.12` uv project for Pomelo Orbit operational commands. Its `src/` package exposes the project-local `pomelo-orbit-cli` entry point; no global installation is required. From this directory, the Makefile is the local automation entry:

```bash
make deps
make run
make run ARGS="database-transfer --help"
make run POMELO_ORBIT_APP__ENV=production ARGS="database-transfer --help"
make check
make test
```

`make deps` syncs the locked dependency groups and installs `pomelo-orbit-cli` into the scripts project environment. `make run` defaults `POMELO_ORBIT_APP__ENV` to `development`; override it on the command line or in the process environment. `make check fix=1` applies Ruff format and lint fixes. Equivalent uv commands from the Orbit repository root:

```bash
uv --directory scripts sync --all-groups --locked
POMELO_ORBIT_APP__ENV=development uv --directory scripts run --locked pomelo-orbit-cli --help
POMELO_ORBIT_APP__ENV=development uv --directory scripts run --locked pomelo-orbit-cli database-transfer --help
```

The CLI locates the Orbit repository, then loads the same configuration layers as the Go service: `configs/config.yaml`, optional `configs/config.<env>.yaml`, `.env` or `.env.<env>`, and process environment overrides. `make run` selects the development profile unless `POMELO_ORBIT_APP__ENV` is already set.

Run the script quality checks with:

```bash
uv --directory scripts run --locked ruff format --check .
uv --directory scripts run --locked ruff check .
uv --directory scripts run --locked mypy
uv --directory scripts run --locked pytest
```
