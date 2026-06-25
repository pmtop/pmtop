# pmtop

> Interactive terminal UI for inspecting Linux **ports**, **processes**, and
> **container associations** — a single pane of glass over `/proc`.

[![CI](https://github.com/pmtop/pmtop/actions/workflows/ci.yml/badge.svg)](https://github.com/pmtop/pmtop/actions/workflows/ci.yml)
[![Release](https://github.com/pmtop/pmtop/actions/workflows/release.yml/badge.svg)](https://github.com/pmtop/pmtop/actions/workflows/release.yml)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`pmtop` unifies `ss`, `lsof`, `htop`, and `docker ps`-style information into one
keyboard-driven TUI. It reads kernel interfaces directly (`/proc/net/*`,
`/proc/<pid>/fd/*`, `/proc/<pid>/cgroup`) with **zero external runtime
dependencies** — no `ss`, `netstat`, `lsof`, or `docker` CLI required.

## Features

- All TCP / UDP / Unix sockets with owning PID, process, user, and container
- Real-time refresh (default 2s, configurable), preserves selection
- Powerful filtering: port ranges, protocols, states, CIDR, fuzzy process match
- Process detail side panel: full cmdline, exe, cwd, cgroup, package owner, FDs
- Container association: Docker / containerd / Podman / CRI-O via cgroup parsing
- Process management: send signals with confirmation dialogs
- Privilege-aware: restricted mode for non-root, opt-in sudo re-launch
- Scriptable CLI: `pmtop list --json`, `pmtop kill`, `pmtop info`
- Accessibility: `NO_COLOR` support + symbol-based state indicators
- Single static binary; packages for apt, dnf, AUR, Homebrew

## Quick start

```bash
# Build
make build
./build/pmtop

# Interactive TUI (run as root for full visibility)
sudo ./build/pmtop

# Non-interactive / scripting
pmtop list --proto tcp --state LISTEN --json
pmtop kill 1234 --signal SIGTERM
pmtop info 1234
```

## Installation

See [INSTALL.md](docs/INSTALL.md) for package-manager install instructions
(apt, dnf, AUR, Homebrew, static binary).

## Configuration

- System: `/etc/pmtop/config.toml`
- User: `~/.config/pmtop/config.toml`
- Precedence: CLI flags > user config > system config > built-in defaults

See `man pmtop.toml(5)` after install.

## Documentation

- `man pmtop(8)` — main tool documentation
- `man pmtop.toml(5)` — configuration file reference
- [docs/](docs/) — additional guides

## Development

```bash
make test        # run tests with coverage
make lint        # golangci-lint (falls back to go vet)
make build-all   # cross-compile amd64 + arm64
```

## License

MIT — see [LICENSE](LICENSE).
