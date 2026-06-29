# pmtop TODO

## v0.2 — Planned Features

### Process Grouping View (rustnet `a` key)
- [ ] Toggle between flat and grouped views with `a` key
- [ ] Group header: `▸/▾` + process name + `(N connections)` + protocol distribution `TCP:10 UDP:2`
- [ ] Expand/collapse groups with `Space` or `←/→`
- [ ] Aggregated stats per group (total connections, state breakdown)
- [ ] Sort groups by connection count or process name

### Inline Vim-Style Filter Syntax (rustnet `/` style)
- [ ] `/port:443` — exact port match
- [ ] `/state:established` — state filter
- [ ] `/process:ssh` — process name filter
- [ ] `/pid:1234` — PID filter
- [ ] `/user:root` — user filter
- [ ] `/proto:tcp` — protocol filter
- [ ] Space = implicit AND: `/port:443 state:established`
- [ ] Regex with `/pattern/` syntax: `/port:/22/` matches 22, 220, 5522
- [ ] Field aliases: `proc:` for `process:`, `src:` for `source:`, `dst:` for `dest:`
- [ ] Keep `f` key as quick preset filter form (complementary to inline syntax)

### Connection Trend Sparkline
- [ ] Sparkline in status bar showing connection count over time
- [ ] Per-state sparklines in summary area
- [ ] Configurable history window (default: 60 samples = 2 minutes at 2s interval)

### Display Toggle: IP/Hostname (`d` key, rustnet style)
- [ ] Resolve IP addresses to hostnames via reverse DNS
- [ ] Toggle between IP and hostname display with `d` key
- [ ] Cache DNS results with TTL to avoid repeated lookups
- [ ] Async resolution (don't block TUI on DNS)

## v0.3+ — Future Ideas

### Tab-based Interface (rustnet style)
- [ ] `1` Overview (current table view)
- [ ] `2` Details (selected connection detail)
- [ ] `3` Interfaces (per-interface RX/TX stats)
- [ ] `4` Graph (traffic charts, top processes)
- [ ] `5` Help

### Bandwidth Monitoring
- [ ] Per-process bandwidth utilization (like nethogs)
- [ ] Requires /proc/net/dev sampling + inode-to-process mapping
- [ ] Rx/Tx columns with decay bar chart

### Export Format Selection
- [ ] `e` key opens format chooser (JSON/CSV/TSV)
- [ ] Configurable default export format

### Config File: Keybinding Customization
- [ ] TOML `[keybindings]` section
- [ ] Override any default key binding
- [ ] Validate bindings on load
