# Product Requirements Document (PRD)
# Linux Port & Process Management Tool — pmtop

**Version**: v1.1
**Date**: 2026-06-25
**Status**: Planning
**Authors**: AI Assistant
**Target Platform**: Linux (x86_64, aarch64/ARM64)
**Distribution**: APT (Debian/Ubuntu), DNF (RHEL/Fedora/CentOS/Rocky/Alma), AUR (Arch), static binary, Homebrew (Linuxbrew)

---

## 1. Document Information

| Attribute | Value |
|-----------|-------|
| Product Name | pmtop |
| Binary Name | `pmtop` |
| Go Module | `github.com/<org>/pmtop` |
| Target Users | Linux system administrators, DevOps engineers, SRE, developers |
| Core Scenarios | Quickly diagnose port usage, resolve conflicts, manage processes, inspect container associations |
| Competitor References | `ss`, `netstat`, `lsof`, `htop`, `lazydocker`, `nethogs` |
| License | MIT |

---

## 2. Project Background & Goals

### 2.1 Background

Existing Linux port management relies on several separate, low-level tools (`ss -tlnp`, `lsof -i`, `docker port`, `ps`). Information is scattered and workflows are tedious. There is a clear need for a **unified, interactive, zero-external-dependency** TUI tool that integrates port, process, and container information into a single interface.

### 2.2 Goals

- **Single pane of glass**: View ports, filter, inspect processes, find containers, and kill processes from one terminal window.
- **Zero external dependencies**: No reliance on `ss`, `netstat`, `lsof`, `docker` CLI. Reads kernel interfaces directly (`/proc/net/*`, `/proc/<pid>/fd/*`).
- **High performance**: Support 10,000+ connections with real-time refresh under 50 MB RSS.
- **Easy distribution**: Single static binary plus native packages via all major package managers.
- **Scriptable**: A CLI (non-TUI) subcommand mode for automation, CI/CD, and testing.
- **Readable documentation**: Auto-generated man pages with rich examples, shell completions for all major shells.

---

## 3. Functional Requirements

### 3.1 FR-01  Port List View

| ID | Requirement | Priority | Acceptance Criteria |
|:--:|---|:--:|:---|
| FR-01-01 | Show all TCP and UDP sockets — LISTEN, ESTABLISHED, and all other states | P0 | Output matches `ss -tunap`; no missing entries |
| FR-01-02 | Show Unix Domain Sockets | P1 | Show path, type (STREAM/DGRAM), state, owning process |
| FR-01-03 | Each row shows: Protocol, Local Address, Local Port, Remote Address, Remote Port, State, PID, Process Name, User, Container Name | P0 | Column widths auto-fit; horizontal scroll when needed |
| FR-01-04 | State color coding (LISTEN=green, ESTABLISHED=blue, TIME_WAIT=yellow, CLOSE_WAIT=red, etc.) **and** symbolic indicators for colorblind accessibility | P1 | Respects `NO_COLOR`; symbols (▶ ..) always shown when color is off |
| FR-01-05 | Sort by column (click header or shortcut to toggle asc/desc) | P1 | Sort by port, PID, process name, state, local address, remote address |
| FR-01-06 | Show network namespace (netns) name per socket | P2 | Resolve `ip netns` names from `/var/run/netns/` |

### 3.2 FR-02  Real-Time Refresh

| ID | Requirement | Priority | Acceptance Criteria |
|:--:|---|:--:|:---|
| FR-02-01 | Auto-refresh every 2 seconds by default, configurable | P0 | Preserves selected row and scroll position across refreshes |
| FR-02-02 | Pause / resume refresh (Space key) | P1 | Clear `[PAUSED]` indicator in status bar while paused |
| FR-02-03 | Configurable refresh interval: 0.5 s, 1 s, 2 s, 5 s, manual-only | P1 | Persisted in config file |
| FR-02-04 | Minimize redraw flicker during refresh | P1 | No visible flicker in TUI |

### 3.3 FR-03  Filtering & Search

| ID | Requirement | Priority | Acceptance Criteria |
|:--:|---|:--:|:---|
| FR-03-01 | Filter by port range: single (80), comma list (80,443), hyphen range (8080-8090) | P0 | All three syntaxes supported, mixed |
| FR-03-02 | Filter by protocol: TCP, UDP, Unix | P0 | Multi-select |
| FR-03-03 | Filter by state: LISTEN, ESTABLISHED, TIME_WAIT, CLOSE_WAIT, etc. | P0 | Multi-select |
| FR-03-04 | Fuzzy search by process name / PID / username | P0 | Real-time (type to filter) |
| FR-03-05 | Filter by local / remote IP address with CIDR (e.g. `192.168.1.0/24`) | P1 | IPv4 and IPv6 |
| FR-03-06 | Filter by container name (fuzzy) | P1 | Fuzzy match |
| FR-03-07 | Combined filters: all active filters ANDed together | P0 | Multiple filter conditions apply simultaneously |
| FR-03-08 | Persistent filter bar at top showing active filters | P1 | Single key clears all filters |
| FR-03-09 | Export filtered view (JSON / CSV) | P2 | Exports only the currently visible rows |

### 3.4 FR-04  Process Information

| ID | Requirement | Priority | Acceptance Criteria |
|:--:|---|:--:|:---|
| FR-04-01 | Select a port row, press Enter or `i` to open process detail panel | P0 | Slide-in side panel or floating overlay |
| FR-04-02 | Process detail shows: PID, PPID, process name, full command line, executable path, working directory, start time, CPU/memory usage, user/group | P0 | Paths are copyable |
| FR-04-03 | Show all file descriptors (including sockets) opened by the process | P1 | Navigable to the corresponding port row |
| FR-04-04 | Show cgroup information; identify whether the process belongs to a container runtime | P1 | Parse `/proc/<pid>/cgroup` (cgroup v1 and v2) |
| FR-04-05 | Identify the owning system package of the executable (`dpkg -S` / `rpm -qf`) | P1 | Show package name and version when available |

### 3.5 FR-05  Container Association

| ID | Requirement | Priority | Acceptance Criteria |
|:--:|---|:--:|:---|
| FR-05-01 | Auto-detect the container runtime for each port (Docker, containerd, Podman, CRI-O) | P0 | Match container ID via `/proc/<pid>/cgroup` (v1 and v2) |
| FR-05-02 | Show container name, short container ID, image name, container status | P0 | Matches `docker ps` / `podman ps` output |
| FR-05-03 | Show container port mappings (internal → external port) | P1 | Parse Docker API `/var/run/docker.sock` or container inspect output |
| FR-05-04 | Navigate to container detail view (read-only: config, network, mounts) | P2 | Read-only display |

### 3.6 FR-06  Process Management

| ID | Requirement | Priority | Acceptance Criteria |
|:--:|---|:--:|:---|
| FR-06-01 | Select a process, press `k` to send a signal | P0 | Signal selection menu appears |
| FR-06-02 | Supported signals: SIGTERM (15), SIGKILL (9), SIGHUP (1), SIGINT (2), SIGUSR1 (10), SIGUSR2 (12) | P0 | SIGTERM is the default selection |
| FR-06-03 | Confirmation dialog before sending a signal, showing process name and PID | P0 | Prevent accidental kills |
| FR-06-04 | Feedback after signal sent: success / failure / permission denied | P0 | Status bar message shown for 3 seconds |
| FR-06-05 | Batch-select multiple processes and send signals to all | P2 | Multi-select mode (Space / Shift+arrows) |
| FR-06-06 | Handle stale selection gracefully: if the selected PID disappears during a refresh while a confirmation dialog is open, show a warning and dismiss the dialog | P1 | Clear message: "Process no longer exists" |

### 3.7 FR-07  Privilege Model

| ID | Requirement | Priority | Acceptance Criteria |
|:--:|---|:--:|:---|
| FR-07-01 | On startup, detect UID. If non-root, show a banner explaining limited view and offer to restart via `sudo` | P0 | "Run `sudo pmtop` for full port view"; offer a key to attempt re-launch with sudo |
| FR-07-02 | Do NOT silently auto-elevate or re-exec | P0 | The user sees the banner and explicitly chooses to re-launch |
| FR-07-03 | Running non-root enters Restricted Mode: only the current user's processes are shown alongside socket info; other users' PIDs are hidden | P0 | Top bar shows `[Restricted Mode]` badge |
| FR-07-04 | CLI flag `--no-elevate` forces current-user mode with no banner (for CI / automation) | P1 | No interactive prompt in CI |
| FR-07-05 | (Optional) Document that `setcap cap_sys_ptrace,cap_net_admin` on the binary allows reading other users' /proc without full root; note that kill still requires root for other users' processes | P2 | Documented in man page and README |

### 3.8 FR-08  Configuration & Persistence

| ID | Requirement | Priority | Acceptance Criteria |
|:--:|---|:--:|:---|
| FR-08-01 | System-level config: `/etc/pmtop/config.toml` | P1 | Read on startup; provides site-wide defaults |
| FR-08-02 | User-level config: `~/.config/pmtop/config.toml` | P1 | Overrides system config; follows XDG base directory spec |
| FR-08-03 | CLI flags override both config files | P1 | Standard precedence: flag > user config > system config > built-in default |
| FR-08-04 | Configurable items: refresh interval, default sort column, color theme, key bindings | P1 | Reload on restart (hot reload is P2) |
| FR-08-05 | Export current view data as JSON / CSV (`e` key) | P2 | Filters are applied to the export |
| FR-08-06 | Environment variable `NO_COLOR` disables all terminal colors (using symbol-based state indicators) | P1 | Follows https://no-color.org |

### 3.9 FR-09  CLI (Non-TUI) Mode

| ID | Requirement | Priority | Acceptance Criteria |
|:--:|---|:--:|:---|
| FR-09-01 | `pmtop list` outputs the port table in plain text (tab-separated for piping) | P0 | Same data source as TUI; filters applicable via flags |
| FR-09-02 | `pmtop list --json` outputs structured JSON | P0 | Machine-readable for scripting and CI |
| FR-09-03 | `pmtop list --csv` outputs CSV with headers | P0 | Columns match TUI table |
| FR-09-04 | `pmtop kill <PID>` sends SIGTERM (default) or a specified signal to a process | P1 | Option `--signal <name|number>` |
| FR-09-05 | `pmtop info <PID>` prints process detail as structured text or JSON | P1 | Same detail as TUI side panel |

### 3.10 FR-10  Accessibility

| ID | Requirement | Priority | Acceptance Criteria |
|:--:|---|:--:|:---|
| FR-10-01 | Respect `NO_COLOR` environment variable | P1 | All colors suppressed; symbolic state indicators enabled |
| FR-10-02 | Provide symbol-based state indicators (`▶` for LISTEN, `●` for ESTABLISHED, etc.) when color is disabled or at user preference | P1 | Toggleable in config (`colorblind_mode = true`) |
| FR-10-03 | All key bindings are configurable | P2 | Mapped in config file |

---

## 4. Non-Functional Requirements

### 4.1 Performance

| ID | Requirement | Metric |
|:--:|---|:---|
| NFR-01 | Startup time | < 500 ms (10,000 connections) |
| NFR-02 | Refresh latency | < 200 ms (full refresh) |
| NFR-03 | Memory footprint | < 50 MB RSS |
| NFR-04 | CPU usage during refresh | < 5% of a single core |

### 4.2 Compatibility

| ID | Requirement | Details |
|:--:|---|:---|
| NFR-05 | Kernel version | Linux 3.10+ (CentOS 7 compatible) |
| NFR-06 | Architectures | x86_64 (amd64), aarch64 (arm64) |
| NFR-07 | Container environment | Runs inside a container with `--pid=host --net=host` |
| NFR-08 | Terminal support | Linux console, SSH, tmux/screen; minimum 80x24 |
| NFR-09 | Cgroup support | cgroup v1 (hierarchical) and cgroup v2 (unified) |

### 4.3 Security

| ID | Requirement | Details |
|:--:|---|:---|
| NFR-10 | Least privilege | Elevation is opt-in and explicit; restricted mode clearly indicated |
| NFR-11 | Signal safety | Mandatory confirmation dialog before any signal is sent |
| NFR-12 | No network requests | Entirely local; never initiates outbound network connections |
| NFR-13 | No telemetry | No analytics, no auto-update pings, no home-phoning of any kind |

### 4.4 Maintainability

| ID | Requirement | Details |
|:--:|---|:---|
| NFR-14 | Code coverage | > 80% (core business logic) |
| NFR-15 | Logging | `--debug` writes structured logs to `~/.local/share/pmtop/debug.log` |
| NFR-16 | Reproducible builds | Use `-trimpath`, `-ldflags` with fixed `BUILD_TIME=$SOURCE_DATE_EPOCH`, CGO disabled |
| NFR-17 | License | MIT |

---

## 5. Technical Architecture

### 5.1 Technology Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Language | Go 1.22+ | Static compilation, single binary, excellent cross-compilation |
| TUI framework | Bubble Tea + Lipgloss + Bubbles | Charm ecosystem, mature and stable |
| Config parsing | `github.com/BurntSushi/toml` | TOML format |
| CLI framework | `github.com/spf13/cobra` | Subcommands, flags, man page generation, completions |
| Logging | standard library `log/slog` | Built-in since Go 1.21 |
| Testing | standard `testing` + `github.com/stretchr/testify` | Unit tests |
| Distribution | Goreleaser + nfpm | Cross-build, packaging, signing, release publishing |

### 5.2 Module Architecture

```
pmtop/
├── cmd/
│   └── pmtop/                  # Main entry point
│       └── main.go
├── internal/
│   ├── app/                    # Bubble Tea application loop
│   │   ├── model.go            # Main Model
│   │   ├── update.go           # Message handling & state updates
│   │   ├── view.go             # Rendering logic
│   │   └── keymap.go           # Default key bindings (configurable)
│   ├── collector/              # Data collection engine
│   │   ├── procfs.go           # /proc/net/* parser
│   │   ├── socket.go           # Socket info types
│   │   ├── process.go          # /proc/<pid> parser
│   │   ├── inode_index.go      # Single-pass inode→PID map builder
│   │   ├── cgroup.go           # cgroup v1/v2 parsing + container runtime detection
│   │   └── docker.go           # Optional Docker API client for rich info
│   ├── filter/                 # Filter engine
│   │   ├── engine.go           # Filter logic
│   │   └── parser.go           # Port range / CIDR parser
│   ├── process/                # Process management
│   │   ├── signal.go           # Signal sending
│   │   ├── info.go             # Process detail collection
│   │   └── pkg_owner.go        # dpkg -S / rpm -qf package ownership
│   ├── ui/                     # UI components
│   │   ├── table.go            # Port list table
│   │   ├── sidebar.go          # Process detail side panel
│   │   ├── filterbar.go        # Active filter display bar
│   │   ├── statusbar.go        # Bottom status bar
│   │   └── dialog.go           # Confirmation dialogs
│   ├── config/                 # Configuration management
│   │   └── config.go           # Layered load: /etc → ~/.config → flags
│   ├── elevate/                # Privilege check & re-launch
│   │   └── elevate.go          # UID detection, sudo re-launch offer
│   └── platform/               # Platform-specific code
│       └── linux.go
├── pkg/                        # Public reusable packages
│   └── netstat/                # Pure Go netstat-like data structures
├── man/                        # Generated man pages (prepare script runs before build)
│   ├── pmtop.8                 # pmtop(8)
│   └── pmtop.toml.5            # pmtop.toml(5)
├── completions/                # Shell completions (generated by cobra)
│   ├── bash/
│   ├── zsh/
│   └── fish/
├── .github/
│   └── workflows/
│       ├── ci.yml              # PR tests: lint, test, build all arches
│       └── release.yml         # Tag → goreleaser → .deb/.rpm/static/AUR/homebrew
├── .goreleaser.yaml            # Goreleaser + nfpm configuration
├── go.mod
├── Makefile
├── README.md
└── LICENSE                     # MIT
```

### 5.3 Core Data Flow

```
┌────────────┐     ┌──────────────┐     ┌─────────────┐
│  Tick Msg  │────▶│  Collector   │────▶│  Socket     │
│  (every 2s)│     │  Engine      │     │  Data Pool  │
└────────────┘     └──────────────┘     └──────┬──────┘
                                                │
                        ┌───────────────────────┘
                        ▼
┌────────────┐     ┌──────────────┐     ┌─────────────┐
│  Render    │◀────│  Filter      │◀────│  Indexer    │
│  (View)    │     │  Engine      │     │  (inode→PID │
└────────────┘     └──────────────┘     │   map)      │
       │                                └──────┬──────┘
       ▼                                       │
┌────────────┐                        ┌───────┴──────┐
│  Docker /  │                        │  Process     │
│  Container │                        │  Enrichment  │
│  Enrich    │                        │  (exe,cgroup,│
│            │                        │   user,etc.) │
└────────────┘                        └──────────────┘
```

### 5.4 Key Algorithm: /proc/net/tcp Parsing + Inode Index

```
// /proc/net/tcp line format:
// sl  local_address rem_address  st tx_queue:rx_queue ... uid ... inode ...
// 0:  0100007F:1F90 00000000:0000 0A 00000000:00000000 ...             12345 ...

// Phase 1: Parse all /proc/net/{tcp,tcp6,udp,udp6,raw,unix} into SocketInfo list
// Phase 2: Build a single inode→PID map by scanning /proc/<pid>/fd/ once
// Phase 3: Join SocketInfo with process info via the inode map
// Phase 4: Enrich with cgroup info, user names, container metadata

// Phase 2 — One-pass inode index (O(total_fds), not O(sockets × pids × fds)):
func buildInodePIDMap() (map[uint64]int, error) {
    index := make(map[uint64]int)
    pids, _ := filepath.Glob("/proc/[0-9]*")
    for _, pidPath := range pids {
        pid, _ := strconv.Atoi(filepath.Base(pidPath))
        fdDir := filepath.Join(pidPath, "fd")
        symlinks, err := filepath.Glob(filepath.Join(fdDir, "*"))
        if err != nil {
            continue  // Skip unreadable (permission denied) gracefully
        }
        for _, symlink := range symlinks {
            target, err := os.Readlink(symlink)
            if err != nil {
                continue
            }
            // socket:[inode] or [0000]:inode for pipe
            if strings.HasPrefix(target, "socket:[") {
                inode := parseInode(target)  // extract uint64 between [ ]
                index[inode] = pid
            }
        }
    }
    return index, nil
}
```

### 5.5 Key Algorithm: Port Range Parser

```
// Supported formats: "80", "80,443", "8080-8090", "80,8080-8090,9000"
func parsePortRange(input string) ([]uint16, error) {
    // Split on commas → for each segment check for '-' range → expand to list
    // Deduplicate + sort
}
```

### 5.6 Container Runtime Detection

```
// Detect container runtime from /proc/<pid>/cgroup:
// cgroup v1: contains "/docker/<container_id>" or "/containerd/<container_id>"
//             or "/libpod/<container_id>" (Podman)
// cgroup v2: "0::/system.slice/docker-<container_id>.scope"
//             or "0::/machine.slice/libpod-<container_id>.scope"

// Parse the container ID from the cgroup path.
// Match against running containers via Docker/Podman socket (if available)
// or /var/lib/docker/containers/ for fallback name resolution.
```

---

## 6. User Interface Design

### 6.1 Main Layout

```
┌────────────────────────────────────────────────────────────────────────────┐
│ pmtop v1.0.0  [root]  Refresh: 2s  │ Filter: TCP, LISTEN, port:8080-8090   │  ← Top status bar
├────────────────────────────────────────────────────────────────────────────┤
│ Proto │ Local Address   │ Port │ Remote Address │ State     │ PID  │ Process    │ Container │
├───────┼─────────────────┼──────┼────────────────┼───────────┼──────┼────────────┼───────────┤
│ TCP ▶ │ 0.0.0.0         │ 8080 │ 0.0.0.0:0      │ LISTEN    │ 1234 │ nginx      │ -         │  ← Selected
│ TCP ● │ 127.0.0.1       │ 8081 │ 0.0.0.0:0      │ LISTEN    │ 5678 │ myapp      │ webapp_1  │
│ TCP ● │ 192.168.1.10    │ 443  │ 0.0.0.0:0      │ LISTEN    │ 9012 │ docker-pro │ traefik   │
│ TCP ● │ 192.168.1.10    │ 22   │ 10.0.0.5:54321 │ ESTAB SHD │ 3456 │ sshd       │ -         │
│ UDP - │ 0.0.0.0         │ 53   │ 0.0.0.0:0      │ -         │ 7890 │ systemd-re │ -         │
├────────────────────────────────────────────────────────────────────────────┤
│ [F1]Help [F2]Filter [F3]Sort [Enter]Detail [k]Kill [q]Quit [Space]Pause    │  ← Bottom key hints
└────────────────────────────────────────────────────────────────────────────┘
```

State symbols (shown alongside or instead of colors when `NO_COLOR` is set):
- `▶` LISTEN
- `●` ESTABLISHED
- `▲` TIME_WAIT
- `▼` CLOSE_WAIT
- `◆` SYN_SENT
- `◀` CLOSING
- `-` no state (UDP, raw)

### 6.2 Process Detail Side Panel (Enter to open)

```
┌──────────────────────────────────────┐
│ Process Detail                       │
├──────────────────────────────────────┤
│ PID:        1234                     │
│ PPID:       1                        │
│ Name:       nginx                    │
│ User:       www-data (33)            │
│ Command:    /usr/sbin/nginx -g daemon│
│ Exe Path:   /usr/sbin/nginx          │
│ CWD:        /var/www                 │
│ Start:      2026-06-25 08:30:15      │
│ CPU:        0.5%   MEM: 12 MB        │
│ Package:    nginx 1.26.0-1 (dpkg)    │
│ Container:  -                        │
│                                      │
│ [o] Open FD list  [k] Send signal  [Esc] Close│
└──────────────────────────────────────┘
```

### 6.3 Signal Selection Dialog

```
┌──────────────────────────────────────┐
│ Send signal to nginx (PID: 1234)     │
├──────────────────────────────────────┤
│   ○ SIGHUP  (1)   — Reload config    │
│   ○ SIGINT  (2)   — Interrupt        │
│   ● SIGTERM (15)  — Graceful stop    │
│   ○ SIGKILL (9)   — Force kill       │
│   ○ SIGUSR1 (10)  — User-defined 1   │
│   ○ SIGUSR2 (12)  — User-defined 2   │
├──────────────────────────────────────┤
│ [Enter] Confirm  [Esc] Cancel        │
└──────────────────────────────────────┘
```

### 6.4 Key Bindings

| Key | Action |
|-----|--------|
| `↑` / `↓` or `k` / `j` | Move selection up/down |
| `PgUp` / `PgDn` | Page up / down |
| `Home` / `End` | Jump to top / bottom |
| `Enter` | Process detail (open side panel) |
| `Esc` | Close side panel / dismiss dialog |
| `Tab` | Switch focus (table ↔ filter bar) |
| `/` | Enter search / filter mode |
| `f` | Open filter panel (protocol, state, port range) |
| `s` | Toggle sort column |
| `k` | Send signal (selected process) |
| `Space` | Pause / resume auto-refresh |
| `r` | Manual refresh |
| `e` | Export current view |
| `q` / `Ctrl+C` | Quit |
| `F1` | Help |

### 6.5 Non-Root Banner

```
┌────────────────────────────────────────────────────────────────────────────┐
│ ⚠ Running without root. Only your own processes are shown.                 │
│   Run `sudo pmtop` for full port and process visibility.                   │
│   Press S to restart with sudo, or any key to continue.                    │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Distribution & Release Pipeline

### 7.1 Build Targets

| Platform | Arch | Format | Install Command |
|----------|------|--------|-----------------|
| Debian 10+ | amd64, arm64 | `.deb` | `sudo apt install ./pmtop_*.deb` |
| Ubuntu 20.04+ | amd64, arm64 | `.deb` | `sudo apt install ./pmtop_*.deb` |
| RHEL 7+ / CentOS 7+ / Rocky / Alma | amd64, arm64 | `.rpm` | `sudo dnf install ./pmtop_*.rpm` |
| Fedora 35+ | amd64, arm64 | `.rpm` | `sudo dnf install ./pmtop_*.rpm` |
| Arch Linux / Manjaro | amd64, arm64 | AUR `pmtop-bin` | `yay -S pmtop-bin` |
| Homebrew on Linux | amd64, arm64 | Formula | `brew install <org>/tap/pmtop` |
| Any Linux | amd64, arm64 | Static binary | `curl -L ... \| tar xz; ./pmtop` |

### 7.2 Goreleaser + nfpm Pipeline

**Single `.goreleaser.yaml`** defines the entire build and release:

```yaml
# .goreleaser.yaml — conceptual structure
builds:
  - id: pmtop
    binary: pmtop
    main: ./cmd/pmtop
    env: [CGO_ENABLED=0]
    goos: [linux]
    goarch: [amd64, arm64]
    flags: [-trimpath]
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

nfpms:
  - id: pmtop
    package_name: pmtop
    homepage: https://github.com/<org>/pmtop
    license: MIT
    formats: [deb, rpm]
    # Man pages & completions installed via nfpm contents
    contents:
      - src: man/pmtop.8
        dst: /usr/share/man/man8/pmtop.8
      - src: man/pmtop.toml.5
        dst: /usr/share/man/man5/pmtop.toml.5
      - src: completions/bash/pmtop
        dst: /usr/share/bash-completion/completions/pmtop
      - src: completions/zsh/_pmtop
        dst: /usr/share/zsh/site-functions/_pmtop
      - src: completions/fish/pmtop.fish
        dst: /usr/share/fish/vendor_completions.d/pmtop.fish

before:
  hooks:
    - make man           # Generate man pages from cobra
    - make completions   # Generate shell completions from cobra

release:
  github:
    owner: <org>
    name: pmtop

aurs:
  - name: pmtop-bin
    homepage: https://github.com/<org>/pmtop
    description: |
      Interactive terminal UI for inspecting Linux ports,
      processes, and container associations.
    license: MIT

brews:
  - name: pmtop
    homepage: https://github.com/<org>/pmtop
    tap:
      owner: <org>
      name: homebrew-tap
```

**GitHub Actions `release.yml`** triggers on `v*` tags:

```yaml
# .github/workflows/release.yml — conceptual structure
on:
  push:
    tags: ["v*"]

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - run: make man && make completions
      - uses: goreleaser/goreleaser-action@v5
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          AUR_KEY: ${{ secrets.AUR_SSH_PRIVATE_KEY }}

  # Smoke-test: install .deb on a Debian container
  test-deb:
    needs: goreleaser
    runs-on: ubuntu-latest
    container: debian:bookworm
    steps:
      - run: apt update && apt install -y ./pmtop_*.deb
      - run: pmtop list --json | head -n 5

  test-rpm:
    needs: goreleaser
    runs-on: ubuntu-latest
    container: rockylinux:9
    steps:
      - run: dnf install -y ./pmtop_*.rpm
      - run: pmtop list --json | head -n 5
```

### 7.3 CI Workflow (`ci.yml`)

```yaml
# .github/workflows/ci.yml — Pull Request validation
on:
  pull_request:
  push:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }
      - uses: golangci/golangci-lint-action@v4

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }
      - run: go test -race -coverprofile=coverage.out ./...
      - uses: codecov/codecov-action@v4

  build-all:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }
      - run: make build-all   # Cross-compile amd64+arm64, verify they link
```

### 7.4 Makefile Targets

```makefile
.PHONY: build build-all test lint clean man completions release

build:
	go build -trimpath -ldflags="-s -w" ./cmd/pmtop

build-all:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o build/pmtop-linux-amd64 ./cmd/pmtop
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o build/pmtop-linux-arm64 ./cmd/pmtop

test:
	go test -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

man:
	go run ./cmd/pmtop man --output-dir man

completions:
	go run ./cmd/pmtop completion bash > completions/bash/pmtop
	go run ./cmd/pmtop completion zsh  > completions/zsh/_pmtop
	go run ./cmd/pmtop completion fish > completions/fish/pmtop.fish

clean:
	rm -rf build/ man/ completions/
```

---

## 8. Man Page & Documentation

### 8.1 Man Pages

| File | Section | Description |
|------|---------|-------------|
| `pmtop.8` | 8 (System Administration) | Main tool documentation |
| `pmtop.toml.5` | 5 (File Formats) | Configuration file reference |

### 8.2 Generation Strategy

man pages are generated from cobra command definitions using `cobra.GenManTree`. To ensure readability:

1. **Rich `Long` descriptions**: The `Long` field of each cobra command becomes the DESCRIPTION section of the man page. Write it in plain, explanatory prose with paragraph breaks.
2. **Explicit `Example` blocks**: cobra's `Example` field maps to the EXAMPLES section. Provide at least 3–5 realistic examples per subcommand.
3. **Match man conventions**: Use `.TH PMTOP 8` header, standard sections (NAME, SYNOPSIS, DESCRIPTION, OPTIONS, EXAMPLES, FILES, ENVIRONMENT, SEE ALSO, BUGS).

### 8.3 Example cobra command with man-friendly descriptions

```go
var listCmd = &cobra.Command{
    Use:   "list [flags]",
    Short: "List ports and connections",
    Long: `List all TCP, UDP, and Unix domain sockets along with their
associated processes and container information.

By default, pmtop list outputs a tab-separated table suitable for
piping through awk, cut, or grep. Use --json or --csv for structured
output that can be consumed by scripts and monitoring tools.

The --filter flag accepts a comma-separated list of conditions:
port:80,443,8080-8090   Filter by port number or range
state:LISTEN,ESTAB       Filter by connection state
proto:tcp,udp            Filter by protocol`,
    Example: `  # List all listening TCP ports as JSON
  pmtop list --proto tcp --state LISTEN --json

  # Watch for changes every 1 second (pipe-friendly)
  pmtop list --interval 1

  # Filter ports 8080-8090 used by processes named "java"
  pmtop list --filter port:8080-8090 --filter proc:java

  # Export full port table as CSV
  pmtop list --csv > ports.csv`,
}
```

---

## 9. Shell Completions

Generated from cobra at build time and packaged in each distribution format.

| Shell | File | Install Path |
|-------|------|--------------|
| Bash | `completions/bash/pmtop` | `/usr/share/bash-completion/completions/pmtop` |
| Zsh | `completions/zsh/_pmtop` | `/usr/share/zsh/site-functions/_pmtop` |
| Fish | `completions/fish/pmtop.fish` | `/usr/share/fish/vendor_completions.d/pmtop.fish` |
| PowerShell | `completions/powershell/pmtop.ps1` | PowerShell profile directory |

Generated via:

```
pmtop completion bash > /usr/share/bash-completion/completions/pmtop
pmtop completion zsh  > /usr/share/zsh/site-functions/_pmtop
pmtop completion fish > /usr/share/fish/vendor_completions.d/pmtop.fish
```

---

## 10. Milestone Plan

| Milestone | Time | Deliverables | Acceptance Criteria |
|-----------|------|-------------|---------------------|
| M1: Core Collection | Week 1–2 | `/proc/net/*` parser + inode index + process association | Unit tests pass; output matches `ss -tlnp` |
| M2: TUI Shell | Week 2–3 | Basic table + navigation + sort + refresh | Executable TUI shows port list |
| M3: Filter System | Week 3–4 | Search bar + port range + protocol/state filters + CIDR | All FR-03 criteria |
| M4: Process & Container | Week 4–5 | Process detail panel + signal sending + cgroup detection + container association | FR-04/05/06 criteria |
| M5: Privilege & Config | Week 5–6 | Non-root detection + sudo restart offer + layered config + export + NO_COLOR | FR-07/08/10 criteria |
| M6: CLI Mode | Week 6 | `pmtop list`, `pmtop kill`, `pmtop info` subcommands | All FR-09 criteria; pipeable output |
| M7: Man Pages & Completions | Week 6–7 | cobra GenManTree, man page packaging, shell completions generation, mkdocs source | Man pages readable and complete; completions work in bash/zsh/fish |
| M8: CI/CD & Packaging | Week 7–8 | `.goreleaser.yaml`, `ci.yml`, `release.yml`, nfpm config, install-smoke tests | CI passing; release publishes .deb/.rpm/static/AUR/homebrew |
| M9: Release | Week 8 | v1.0.0 GitHub Release + install docs + AUR submission | All packages installable on target distros |

---

## 11. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| `/proc` format varies across kernel versions | High | Support 3.10+ common formats; graceful degradation on unrecognized fields |
| Docker socket `/var/run/docker.sock` not accessible | Medium | Detect socket readability; hide container detail when unavailable; cgroup-based detection still works |
| High connection count (>50k) performance | Medium | Single-pass inode index; goroutine-parallel pid scanning; debounce UI updates |
| Docker SDK CGO dependency breaks static build | Medium | Avoid official Docker SDK; use raw HTTP calls to Docker Engine API (pure Go, no CGO) |
| Package manager review delays (Debian official repo, Fedora COPR) | Low | Static binary + self-hosted APT/YUM repos are available immediately; official inclusion follows |
| Cgroup format differences (v1 vs v2, systemd slices) | Medium | Parse both formats; fall back to container ID from cgroup path even without runtime socket |
| CI token rotation / AUR SSH key management | Low | Document key setup in CONTRIBUTING.md; use GitHub Actions secrets |

---

## 12. Appendix

### 12.1 References

- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/lipgloss` — Styling
- `github.com/charmbracelet/bubbles` — Components (table, textinput, viewport)
- `github.com/spf13/cobra` — CLI framework, man page generation, completions
- `github.com/goreleaser/goreleaser` — Release automation
- `github.com/goreleaser/nfpm` — Native package building (deb/rpm/apk)
- `github.com/BurntSushi/toml` — TOML config parsing

### 12.2 Related Projects

- `github.com/nicolaka/netshoot` — Network diagnostic container (reference for tool selection)
- `github.com/jesseduffield/lazydocker` — Docker TUI (UI design reference)
- `github.com/derailed/k9s` — Kubernetes TUI (key binding design reference)
- `github.com/gsamokovarov/jump` — Bubble Tea TUI file navigator

### 12.3 Connection State Codes (/proc/net/tcp)

| Hex | State |
|-----|-------|
| 01 | ESTABLISHED |
| 02 | SYN_SENT |
| 03 | SYN_RECV |
| 04 | FIN_WAIT1 |
| 05 | FIN_WAIT2 |
| 06 | TIME_WAIT |
| 07 | CLOSE |
| 08 | CLOSE_WAIT |
| 09 | LAST_ACK |
| 0A | LISTEN |
| 0B | CLOSING |

---

**End of Document**
