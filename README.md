# DevDocs CLI

> WIP, a marriage between my awful go skills and LLM hallucinations

A command-line interface for browsing [DevDocs](https://devdocs.io) documentation offline. This tool allows you to:

- Download documentation for multiple programming languages and frameworks
- Browse documentation offline
- Search across all installed documentation
- View documentation entries in your terminal

## Requirements

- `lynx` browser. Currently hardcoded, more configurable in the future. Maybe even a builtin viewer 🤷

## Features

- Interactive TUI interface
- Documentation version management
- Cached documentation for offline access
- HTML content rendering in terminal

## Installation

```bash
go install github.com/bart-jaskulski/ddc@latest
```

## Usage

### List installed documentation
```bash
ddc list
```

### Search documentation
```bash
ddc search <query>
```

### Browse documentation
```bash
ddc view <docset>
```

Documentation is cached in `~/.local/share/devdocs` by default.

## License

MIT
