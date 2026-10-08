# Pomelo Orbit CLI

`scripts/` is the Python `>=3.12` uv project for Pomelo Orbit operational commands. Its `src/` package exposes the project-local `pomelo-orbit-cli` entry point; no global installation is required. From this directory, the Makefile is the local automation entry:

```bash
make deps
make run
make run -- --help
make run -- cert --help
make run -- ulid -n 3
make run POMELO_ORBIT_APP__ENV=production -- --version
make check
make test
```

`make deps` syncs the locked dependency groups and installs `pomelo-orbit-cli` into the scripts project environment. `make run` defaults `POMELO_ORBIT_APP__ENV` to `development`; override it on the command line or in the process environment. For arguments containing spaces or `=`, use `make run ARGS='cert new -n app.localhost --cert-dir "./local certs"'`. `make check fix=1` applies Ruff format and lint fixes. Equivalent uv commands from the Orbit repository root:

```bash
uv --directory scripts sync --all-groups --locked
POMELO_ORBIT_APP__ENV=development uv --directory scripts run --locked pomelo-orbit-cli --help
POMELO_ORBIT_APP__ENV=development uv --directory scripts run --locked pomelo-orbit-cli --version
```

The CLI locates the Orbit repository, then loads the same configuration layers as the Go service: `configs/config.yaml`, optional `configs/config.<env>.yaml`, `.env` or `.env.<env>`, and process environment overrides. `make run` selects the development profile unless `POMELO_ORBIT_APP__ENV` is already set.

## Commands

`cert` and `ulid` are top-level commands:

```bash
uv --directory scripts run --locked pomelo-orbit-cli cert new -n app.localhost --cert-dir ./certs
uv --directory scripts run --locked pomelo-orbit-cli cert check -n app.localhost --cert-dir ./certs
uv --directory scripts run --locked pomelo-orbit-cli ulid -n 3
```

`cert` requires `mkcert`; run `mkcert -install` once to set up the local CA. `new` writes the certificate and private key to `<cert-dir>/<domain>.pem`, overwriting an existing file. `check` inspects the certificate and trust chain and connects to `<domain>:443`. See the [certificate management guide](../docs/guides/certificate-management.md) for upload and Gateway usage.

`ulid` prints one identifier per line. It generates one by default; `-n` / `--count` sets the quantity.

Standalone scripts such as `manage.py`, `install.py`, and `version-calc.py` remain available through uv. Remote deployment commands in `manage.py` use `scripts/.env`.

Run the script quality checks with:

```bash
uv --directory scripts run --locked ruff format --check .
uv --directory scripts run --locked ruff check .
uv --directory scripts run --locked mypy
uv --directory scripts run --locked pytest
```

## Release versions

From the repository root:

```bash
task version            # calculate from Git history without writing files
task version -- --apply # write the calculated version to release metadata
```

The calculator walks all commits reachable from `HEAD`, oldest first, starting
at `0.0.0`. A subject starting with `feat` (case-insensitive) increases the minor
version and resets the patch version. Every other commit, including fixes,
refactors, documentation and merge commits, increases the patch version. The
major version stays at `0`. Uncommitted changes do not affect the calculation.
Complete Git history is required; for a shallow clone, fetch it with
`git fetch --unshallow` first.

`task version` prints the candidate version and leaves all metadata untouched.
`task version -- --apply` updates `VERSION`, `configs/config.yaml` (`app.version`),
`.env.example` and `web/package.json`, validating every target before writing.
It does not stage, commit or tag the changes.

`VERSION` records the selected release version. Package names, Docker release
tags and release CI continue to use that file, so run `task version -- --apply` when
preparing a release. The Git history calculation can advance after further
commits while the stored release version remains unchanged.
