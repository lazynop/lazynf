# Changelog

All notable changes to Vellum are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- `vellum list` now renders a multi-column color-coded grid on a TTY, with installed fonts marked by a green ✓ + release tag and imported fonts marked yellow. Pipe / redirected output stays plain one-per-line for scripts.
- `vellum list --installed` renders a bordered table on a TTY (font / release / installed at). Pipe / redirected output is `<name>\t<release>` per line for scripts.

### Added
- `vellum update [<font>...]` re-downloads installed fonts whose recorded release no longer matches the upstream catalog (or were imported with the `imported` sentinel). With no arguments, updates everything stale; with `--force`, refreshes even fonts already at the latest release.
- `vellum import [<font>...]` adopts Nerd Fonts already present on disk (e.g. installed by getnf or manually) into Vellum's state manifest. With `--detect`, hashes the on-disk files against the latest release to record the actual upstream tag; without `--detect`, marks the release as `imported` (a future `vellum update` will refresh it). `--all` scans the font dir and imports every subdirectory whose name matches a catalog entry.

## [0.1.0] - YYYY-MM-DD

### Added
- `vellum install <font>...` — download and install one or more Nerd Fonts on Linux, with progress bars and per-font conflict detection.
- `vellum list [--installed]` — list available fonts from the upstream catalog or only those installed by Vellum.
- `vellum search <query>` — case-insensitive substring search over the catalog.
- `vellum cache clean` — clear the catalog cache and any kept archives (idempotent).
- Best-effort batch installs: per-font failures are reported in a final summary and do not abort the run.
- Strict conflict policy: refuses to overwrite directories Vellum did not create unless `--force` is passed.
- Tag-invalidated catalog cache (mirrors `getnf`'s strategy): one GitHub API call per command in the steady state.
- Authentication chain: `GITHUB_TOKEN` env, then `gh auth token`, then unauthenticated.
- Automatic `fc-cache -f` after install (Linux), with `--no-cache-refresh` opt-out.
