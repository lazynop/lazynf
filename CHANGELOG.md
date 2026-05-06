# Changelog

All notable changes to lazynf are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- `lazynf list` now renders a multi-column color-coded grid on a TTY, with installed fonts marked by a green ✓ + release tag and imported fonts marked yellow. Pipe / redirected output stays plain one-per-line for scripts.
- `lazynf list` catalog grid no longer prints inline release tags. Installed fonts are highlighted by color (green = installed, yellow = imported), and a one-line legend below the grid points to `lazynf list --installed` for full details.
- `lazynf list --installed` renders a bordered table on a TTY (font / release / installed at). Pipe / redirected output is `<name>\t<release>` per line for scripts.

### Added
- `lazynf remove <font>...` — uninstalls fonts. By default, deletes on-disk files for fonts installed via lazynf, and only de-adopts (manifest-only) fonts that were imported from elsewhere. With `--purge`, also deletes the on-disk directory of imported fonts. `--no-cache-refresh` opts out of the final `fc-cache`.
- `lazynf update [<font>...]` re-downloads installed fonts whose recorded release no longer matches the upstream catalog (or were imported with the `imported` sentinel). With no arguments, updates everything stale; with `--force`, refreshes even fonts already at the latest release.
- `lazynf import [<font>...]` adopts Nerd Fonts already present on disk (e.g. installed by getnf or manually) into lazynf's state manifest. With `--detect`, hashes the on-disk files against the latest release to record the actual upstream tag; without `--detect`, marks the release as `imported` (a future `lazynf update` will refresh it). `--all` scans the font dir and imports every subdirectory whose name matches a catalog entry.

## [0.1.0] - YYYY-MM-DD

### Added
- `lazynf install <font>...` — download and install one or more Nerd Fonts on Linux, with progress bars and per-font conflict detection.
- `lazynf list [--installed]` — list available fonts from the upstream catalog or only those installed by lazynf.
- `lazynf search <query>` — case-insensitive substring search over the catalog.
- `lazynf cache clean` — clear the catalog cache and any kept archives (idempotent).
- Best-effort batch installs: per-font failures are reported in a final summary and do not abort the run.
- Strict conflict policy: refuses to overwrite directories lazynf did not create unless `--force` is passed.
- Tag-invalidated catalog cache (mirrors `getnf`'s strategy): one GitHub API call per command in the steady state.
- Authentication chain: `GITHUB_TOKEN` env, then `gh auth token`, then unauthenticated.
- Automatic `fc-cache -f` after install (Linux), with `--no-cache-refresh` opt-out.
