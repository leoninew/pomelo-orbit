# Version Calculator

`version-calc.py` derives the release version from Git history. A `feat` commit
increments the minor version; every other commit increments the patch version.

```bash
python scripts/version-calc.py
python scripts/version-calc.py --apply
python scripts/version-calc.py --quiet --apply-amend
```

`--apply` writes the calculated version to `VERSION`, `configs/config.yaml`,
`.env.example`, and `web/package.json`. It creates a lightweight `v<version>`
tag only when the worktree was clean before the update; it does not create a
commit.

`--apply-amend` is for adding release metadata to the latest local commit. It
requires all of the following before changing files:

- The current branch tracks an upstream branch.
- `HEAD` is ahead of that upstream, so the commit being amended is unpushed.
- Any existing calculated `v<version>` tag points to the current local `HEAD`.

It writes and stages the four version files, runs `git commit --amend --no-edit`
with a path restriction when those files changed, then creates the lightweight
release tag on the final HEAD. It never pushes the amended commit or tag. An
existing local tag on the old HEAD is moved to the amended commit. Other staged,
unstaged, and untracked files are neither staged nor included in the amendment.
If the metadata already has the calculated version, it skips the amend and
creates the tag on the existing HEAD when needed.
