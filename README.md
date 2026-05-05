# lazynf

Install [Nerd Fonts](https://www.nerdfonts.com/font-downloads) from your terminal.

> **Status:** early development. MVP supports `install`, `list`, `search`, `cache clean` on Linux.

## Install

```bash
go install github.com/lazynop/lazynf@latest
```

## Usage

```bash
lazynf install JetBrainsMono FiraCode    # install one or more fonts
lazynf list                              # show available fonts
lazynf list --installed                  # show what's installed
lazynf search mono                       # find fonts by substring
lazynf cache clean                       # clear catalog cache
```

Run `lazynf --help` for full options.

## Build from source

```bash
git clone https://github.com/lazynop/lazynf
cd lazynf
just build
./bin/lazynf --help
```

Requires Go 1.25+ and (optionally) [`just`](https://github.com/casey/just).

## Status

MVP — Linux only, install/list/search/cache subcommands. macOS, Windows, `remove`, `update`, and the interactive TUI are planned.

## License

MIT
