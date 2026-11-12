# sprint-ledger-browser

A terminal browser for a project's `.ai/` sprint ledger. It reads `ledger.yaml` or `ledger.tsv`, shows sprints in a navigable table, and displays the associated `SPRINT-*.md` content in a detail pane. The executable is named `sprint-ledger`.

## Build and run

Install Go 1.25.5 or newer and Make, then run from this repository:

```sh
make build
./sprint-ledger --help
./sprint-ledger /path/to/project
```

The project path must contain the `.ai/` sprint data. With no argument, the browser uses the current directory if it contains `.ai/`.

Use `j`/`k` to navigate, Enter or Tab to change focus, Esc to go back, `r` to reload, and `q` to quit. Run the interface in an interactive terminal.

## Development

```sh
make test
```

`make install` builds and installs the executable to `~/.local/bin` by default; override `PREFIX` to choose another installation prefix. The Go sources and tests describe ledger parsing and supported layouts.
