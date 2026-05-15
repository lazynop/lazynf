# Changelog

All notable changes to lazynf are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `lazynf install <font>...` — download and install one or more Nerd Fonts on Linux and macOS, with progress bars and per-font conflict detection. Supports `--force`, `--dest`, `--keep-archive`, and `--no-cache-refresh`.
- `lazynf import [<font>...]` — adopt Nerd Fonts already present on disk (e.g. installed by getnf or manually) into lazynf's state manifest. With `--detect`, hashes the on-disk files against the latest release to record the actual upstream tag; without `--detect`, marks the release as `imported` (a future `lazynf update` will refresh it). `--all` scans the font dir and imports every subdirectory whose name matches a catalog entry.
- `lazynf update [<font>...]` — re-downloads installed fonts whose recorded release no longer matches the upstream catalog (or were imported with the `imported` sentinel). With no arguments, updates everything stale; with `--force`, refreshes even fonts already at the latest release. Reuses the resolved catalog (no double GitHub call). Supports `--keep-archive`.
- `lazynf remove [<font>...]` — uninstalls fonts. By default, deletes on-disk files for fonts installed via lazynf, and only de-adopts (manifest-only) fonts that were imported from elsewhere. With `--purge`, also deletes the on-disk directory of imported fonts. With `--all`, removes every font in the manifest at once; on a TTY the user is prompted for confirmation, while non-interactive use requires `--yes` (also accepted as `-y`). `--no-cache-refresh` opts out of the final `fc-cache`.
- `lazynf list [--installed]` — multi-column color-coded grid on a TTY (green ✓ + release tag for installed, yellow for imported), with a one-line legend pointing to `--installed` for full details. `--installed` renders a bordered table (font / release / installed at). Pipe / redirected output stays plain (one font per line, or `<name>\t<release>` for `--installed`) for scripts.
- `lazynf search <query>` — case-insensitive substring search over the catalog.
- `lazynf cache clean` — clear the catalog cache and any kept archives (idempotent).
- `lazynf cache refresh` — force a fresh fetch of the Nerd Fonts catalog, bypassing the freshness check.
- Interactive TUI (`lazynf` with no arguments on a TTY): single-list + detail layout, multi-select batch operations with `space`, live log pane with persisted failure log under `$XDG_STATE_HOME/lazynf/tui.log`, doctor diagnostics with actionable fixes. Press `?` for the key map. CLI sub-commands remain available and unchanged.
- Interactive conflict resolution on install/import: when a font is already tracked as `imported` in the manifest, or when a font directory exists on disk without a manifest entry, the TUI opens a modal offering Skip / Force (and Adopt for the on-disk case); the CLI prints a clear `conflict (use --force to override)` message and exits non-zero. Pass `--force` to bypass the prompt and overwrite silently.
- `lazynf doctor` — diagnoses lazynf's environment and state. Reports on font directories, fc-cache availability, GitHub auth source, manifest integrity, catalog cache freshness, and orphan directories. No network calls and no automatic fixes — points to the existing commands (`list`, `import`, `update`, `remove`) that resolve each issue.
- Shell completion via `lazynf completion {bash|zsh|fish|powershell}`. Tab completion suggests font names dynamically: catalog entries for `install`, manifest entries for `update`/`remove`, orphan candidates for `import`. No network calls — completion silently returns no suggestions when the catalog cache is absent (run `lazynf list` to populate).
- Best-effort batch installs: per-font failures are reported in a final summary and do not abort the run.
- Strict conflict policy: refuses to overwrite directories lazynf did not create unless `--force` is passed.
- Tag-invalidated catalog cache (mirrors `getnf`'s strategy): one GitHub API call per command in the steady state.
- Authentication chain: `GITHUB_TOKEN` env, then `gh auth token`, then unauthenticated.
- Automatic `fc-cache -f` after install on Linux, with `--no-cache-refresh` opt-out. No-op on macOS by design.
