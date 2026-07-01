run lint for scripts

```bash
python -m mypy scripts/
python -m ruff check scripts/ --fix
python -m ruff format scripts/
```
