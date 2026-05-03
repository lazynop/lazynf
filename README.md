# Vellum

Install [Nerd Fonts](https://www.nerdfonts.com/font-downloads) from your terminal.

> **Status:** early development. MVP supports `install`, `list`, `search`, `cache clean` on Linux.

## Install

```bash
go install github.com/lazynop/vellum@latest
```

## Usage

```bash
vellum install JetBrainsMono FiraCode    # install one or more fonts
vellum list                              # show available fonts
vellum list --installed                  # show what's installed
vellum search mono                       # find fonts by substring
vellum cache clean                       # clear catalog cache
```

Run `vellum --help` for full options.

## Build from source

```bash
git clone https://github.com/lazynop/vellum
cd vellum
just build
./bin/vellum --help
```

Requires Go 1.25+ and (optionally) [`just`](https://github.com/casey/just).

## Status

MVP — Linux only, install/list/search/cache subcommands. macOS, Windows, `remove`, `update`, and the interactive TUI are planned.

## License

MIT
