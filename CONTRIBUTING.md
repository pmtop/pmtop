## Contributing

### Prerequisites

- Go 1.22+
- Linux (pmtop only runs on Linux)
- `make` for build targets

### Development Workflow

1. Fork the repo and create a feature branch from `main`.
2. Run `make lint` and `make test` before committing.
3. Write tests for new code (coverage > 80% is expected on core packages).
4. Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/).
5. Open a pull request — CI must pass before merge.

### Running Tests

```bash
make test          # unit tests
make test-cover    # with coverage report
make lint          # go vet + staticcheck
make build-all     # cross-compile amd64 + arm64
```

### Project Structure

- `cmd/pmtop/` — CLI entry point (cobra commands)
- `internal/` — all library code
  - `collector/` — procfs parsing, cgroup & container detection
  - `app/` — Bubble Tea TUI model/view/update
  - `filter/` — interactive and CLI filter engine
  - `process/` — signal handling, package lookup
  - `config/` — layered TOML configuration
  - `elevate/` — privilege detection & sudo re-launch
  - `export/` — JSON/CSV/TSV output
  - `ui/` — rendering helpers (table, color, panels)
  - `netstat/` — protocol/state types
  - `version/` — build info
- `docs/` — man pages, install guide

### Code of Conduct

Be respectful. This is a small open-source project — all contributions are welcome.
