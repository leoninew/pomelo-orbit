# Version Calculator

`version-calc.py` derives the release version from Git history. A `feat` commit
increments the minor version; every other commit increments the patch version.

```bash
uv --directory scripts run version-calc.py
uv --directory scripts run version-calc.py --apply
```

`--apply` writes the calculated version to `VERSION`, `configs/config.yaml`,
`.env.example`, and `web/package.json`. The script only reads Git history to
derive the version; it never stages files, amends commits, or creates tags.
