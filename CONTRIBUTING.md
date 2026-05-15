# Contributing to lazynf

## Build + test

```bash
just build       # compiles bin/lazynf with version metadata
just test        # runs all tests
just check       # fmt-check + vet + test (full local gate)
just coverage    # generates coverage.html for internal/engine + internal/tui/components
```

## Commits

- Conventional Commit style: `feat:`, `fix:`, `refactor:`, `test:`, `chore:`, `docs:`, `perf:`, `ci:`
- One logical change per commit. Rebase or amend before pushing if needed.
- **No AI attribution** anywhere in the repository: no `Co-Authored-By: Claude`, no "Generated with X" footers, no `// AI-generated` comments. This applies to commits, code, and documentation alike.

## Branching

- Feature work goes in a `feat/<short-name>` branch (or `fix/`, `chore/`, etc.).
- For PR review, push the branch explicitly: `git push -u origin feat/X:feat/X`.
- For solo fast-forward into main: `git push origin feat/X:main` (only when CI is green and the maintainer is OK with skipping review).
- `git push` requires explicit confirmation by the maintainer in every case. Never `--force` against `main`.

## Coverage target

- `internal/engine/` and `internal/tui/components/` must stay at or above **80%** total line coverage.
- CI enforces this in `.github/workflows/check.yml`.
- New code that adds behavior must include tests. Refactors must preserve (or raise) the prior coverage level.
- `cmd/*`, `internal/tui/app/`, `internal/ui/` are intentionally out of the coverage gate: thin glue or legacy.

## Style

- English everywhere in committed code: identifiers, comments, doc comments, error/log/UI strings, test descriptions.
- `gofmt -w .` clean before commit (CI gates on `fmt-check`).
- Doc comments on every exported identifier (package, type, function, constant).
- Prefer `min`/`max` builtins (Go 1.21+) over hand-rolled helpers.
- Tests use `github.com/stretchr/testify` (`require` for hard assertions). Avoid time-based assertions; prefer deterministic channel/state synchronisation.
