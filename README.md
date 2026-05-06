# lazynf

Install [Nerd Fonts](https://www.nerdfonts.com/font-downloads) from your terminal.

> **Status:** early development. MVP supports `install`, `import`, `update`, `list`, `search`, `cache clean` on Linux. No release tagged yet — build from source.

## Install

```bash
go install github.com/lazynop/lazynf@latest
```

## Usage

```bash
lazynf install JetBrainsMono FiraCode    # install one or more fonts
lazynf import --all                      # adopt fonts already on disk into lazynf state
lazynf import JetBrainsMono --detect     # adopt and hash-match against upstream tag
lazynf update                            # refresh stale or imported fonts
lazynf update --force                    # refresh even fonts already at the latest tag
lazynf list                              # color grid of available fonts (TTY) / plain on pipe
lazynf list --installed                  # bordered table of installed fonts
lazynf search mono                       # find fonts by substring
lazynf cache clean                       # clear catalog cache and kept archives
```

Global flags: `-q/--quiet` (errors only), `-v/--verbose` (extra diagnostics on stderr).

Run `lazynf --help` for full options.

### Authentication

`lazynf` makes one GitHub API call per command in the steady state (cached against the upstream release tag). It picks up credentials from, in order: `GITHUB_TOKEN`, `gh auth token`, then unauthenticated. Anonymous use is fine for occasional installs; authenticate to avoid GitHub's anonymous rate limits on heavy use.

## Build from source

```bash
git clone https://github.com/lazynop/lazynf
cd lazynf
just build
./bin/lazynf --help
```

Requires Go 1.25+ and (optionally) [`just`](https://github.com/casey/just).

## Status

MVP — Linux only. Implemented: `install`, `import`, `update`, `list`, `search`, `cache clean`. Planned: macOS and Windows support, `remove`, shell completion, and an interactive TUI.

## License

MIT
