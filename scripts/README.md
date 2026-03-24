run lint for scripts

```bash
uv run --project backend python -m mypy scripts/
uv run --project backend python -m ruff check scripts/ --fix
uv run --project backend python -m ruff format scripts/
```
