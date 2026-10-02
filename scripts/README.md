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

Database reset, SQLite data copy, and native database backups use `database-ops.py` and `scripts/.env`:

```bash
./scripts/database-ops.sh reset                 # show the reset plan
./scripts/database-ops.sh reset --no-dry-run    # execute the reset
./scripts/database-ops.sh copy-from-sqlite                # show the copy plan
./scripts/database-ops.sh copy-from-sqlite --no-dry-run   # copy data into the migrated schema
./scripts/database-ops.sh backup                 # write a PostgreSQL .dump to scripts/backup
./scripts/database-ops.sh backup --engine mysql  # write a MySQL .sql to scripts/backup
```

Backups use the database DSN in `scripts/.env`. MySQL backup requires `MYSQL_DSN`; PostgreSQL backup uses `POSTGRES_DSN`. Execution needs a native dump client, a matching local database container, or an already available Docker client image. The command does not download an image.

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
task version        # calculate from Git history without writing files
task version:apply  # write the calculated version to release metadata
```

The calculator walks all commits reachable from `HEAD`, oldest first, starting
at `0.0.0`. A subject starting with `feat` (case-insensitive) increases the minor
version and resets the patch version. Every other commit, including fixes,
refactors, documentation and merge commits, increases the patch version. The
major version stays at `0`. Uncommitted changes do not affect the calculation.
Complete Git history is required; for a shallow clone, fetch it with
`git fetch --unshallow` first.

`task version` prints the candidate version and leaves all metadata untouched.
`task version:apply` updates `VERSION`, `configs/config.yaml` (`app.version`),
`.env.example` and `web/package.json`, validating every target before writing.
It does not stage, commit or tag the changes.

`VERSION` records the selected release version. Package names, Docker release
tags and release CI continue to use that file, so run `task version:apply` when
preparing a release. The Git history calculation can advance after further
commits while the stored release version remains unchanged.
