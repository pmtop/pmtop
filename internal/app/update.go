package app

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/pmtop/pmtop/internal/filter"
	"github.com/pmtop/pmtop/internal/process"
	"github.com/pmtop/pmtop/internal/ui"
)

// Update handles all messages: window resizing, refresh ticks, and key events.
// Key handling depends on the current interaction mode.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tbl.SetWidth(msg.Width)
		h := msg.Height - 2
		if h < 3 {
			h = 3
		}
		m.tbl.SetHeight(h)
		m.tbl.SetColumns(ui.BuildColumns(msg.Width))
		return m, nil

	case tickMsg:
		cmds := []tea.Cmd{tickCmd(m.interval)}
		if !m.paused {
			m.refresh()
		}
		return m, tea.Batch(cmds...)

	case refreshMsg:
		m.refresh()
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeSearch:
			return m.updateSearch(msg)
		case modeFilter:
			return m.updateFilterForm(msg)
		case modeDetail:
			return m.updateDetail(msg)
		case modeSignal:
			return m.updateSignal(msg)
		default:
			return m.updateTable(msg)
		}
	}

	return m, nil
}

// updateTable handles keys in the default table-navigation mode.
func (m Model) updateTable(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Export and quit work even on an empty table; navigation/search/filter
	// are skipped when there are no rows.
	empty := len(m.socks) == 0
	switch {
	case keyMatches(msg, m.keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case keyMatches(msg, m.keys.Export):
		m.doExport()
		return m, nil
	case empty && !keyMatches(msg, m.keys.Search) && !keyMatches(msg, m.keys.Filter):
		return m, nil

	case keyMatches(msg, m.keys.Pause):
		m.paused = !m.paused
		if m.paused {
			m.setStatus("[PAUSED]", 0)
		} else {
			m.setStatus("resumed", 2*time.Second)
		}
		return m, nil

	case keyMatches(msg, m.keys.Refresh):
		m.refresh()
		m.setStatus("refreshed", time.Second)
		return m, nil

	case keyMatches(msg, m.keys.Sort):
		m.sortKey = m.sortKey.next()
		m.applySort()
		m.setStatus("sort: "+m.sortKey.String()+" asc", 2*time.Second)
		return m, nil

	case keyMatches(msg, m.keys.SortDir):
		m.sortAsc = !m.sortAsc
		m.applySort()
		dir := "asc"
		if !m.sortAsc {
			dir = "desc"
		}
		m.setStatus("sort: "+m.sortKey.String()+" "+dir, 2*time.Second)
		return m, nil

	case keyMatches(msg, m.keys.Search):
		m.mode = modeSearch
		m.searchInput.Reset()
		m.searchInput.Focus()
		return m, textinput.Blink

	case keyMatches(msg, m.keys.Filter):
		m.enterFilterForm()
		return m, textinput.Blink

	case keyMatches(msg, m.keys.Escape):
		// In table mode Esc clears all active filters (FR-03-08).
		if !m.filt.IsEmpty() {
			m.filt = filter.Filter{}
			m.rebuild()
			m.clampCursor()
			m.setStatus("filters cleared", 2*time.Second)
		}
		return m, nil

	case keyMatches(msg, m.keys.Up):
		m.tbl.MoveUp(1)
		return m, nil
	case keyMatches(msg, m.keys.Down):
		m.tbl.MoveDown(1)
		return m, nil
	case keyMatches(msg, m.keys.PageUp):
		m.tbl.MoveUp(m.tbl.Height())
		return m, nil
	case keyMatches(msg, m.keys.PageDn):
		m.tbl.MoveDown(m.tbl.Height())
		return m, nil
	case keyMatches(msg, m.keys.Home):
		m.tbl.GotoTop()
		return m, nil
	case keyMatches(msg, m.keys.End):
		m.tbl.GotoBottom()
		return m, nil

	case keyMatches(msg, m.keys.Enter):
		m.openDetail()
		return m, nil
	case keyMatches(msg, m.keys.Kill):
		m.openSignal()
		return m, nil
	case keyMatches(msg, m.keys.Help):
		m.setStatus("not available yet", time.Second)
		return m, nil
	}
	return m, nil
}

// updateDetail handles the process detail side panel: Esc closes it.
func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape, tea.KeyEnter:
		m.mode = modeTable
		m.detail = nil
		return m, nil
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit
	}
	// 'k' inside detail could be reused to send a signal (FR-04-02 hints).
	if keyMatches(msg, m.keys.Kill) && m.detail != nil && m.detail.pid > 0 {
		m.signal = &SignalState{pid: m.detail.pid, name: m.detail.proc.Name, sel: defaultSignalIndex()}
		m.mode = modeSignal
		m.detail = nil
		return m, nil
	}
	return m, nil
}

// updateSignal handles the signal-selection dialog and its confirmation step
// (FR-06-01..04). Up/Down choose, Enter confirms (with a confirmation dialog),
// Esc cancels.
func (m Model) updateSignal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.signal == nil {
		m.mode = modeTable
		return m, nil
	}
	switch msg.Type {
	case tea.KeyEscape:
		if m.signal.confirm {
			m.signal.confirm = false // back to selection
			m.signal.result = ""
		} else {
			m.mode = modeTable
			m.signal = nil
		}
		return m, nil
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyUp, tea.KeyDown, tea.KeyRunes:
		if keyMatches(msg, m.keys.Up) && !m.signal.confirm {
			if m.signal.sel > 0 {
				m.signal.sel--
			}
			return m, nil
		}
		if keyMatches(msg, m.keys.Down) && !m.signal.confirm {
			if m.signal.sel < len(process.Signals)-1 {
				m.signal.sel++
			}
			return m, nil
		}
	}

	if m.signal.confirm {
		// Confirmation step: Enter sends, Esc (handled above) cancels.
		if msg.Type == tea.KeyEnter {
			m.sendCurrentSignal()
			res := m.signal.result
			m.setStatus(res, 3*time.Second)
			m.mode = modeTable
			m.signal = nil
			// Trigger a refresh so the table reflects the signal effect.
			return m, refreshCmd()
		}
		return m, nil
	}

	// Selection step: Enter opens the confirmation dialog.
	if msg.Type == tea.KeyEnter {
		m.signal.confirm = true
		return m, nil
	}
	return m, nil
}

// updateSearch handles the '/' free-text search input (real-time filter).
func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter, tea.KeyEscape:
		// Exit search mode, keeping the typed filter applied.
		m.searchInput.Blur()
		m.mode = modeTable
		m.clampCursor()
		return m, nil
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.filt.Text = m.searchInput.Value()
	if m.full != nil {
		m.rebuild()
		m.clampCursor()
	}
	return m, cmd
}

// enterFilterForm opens the filter form, pre-populating inputs from the active
// filter and saving a draft to restore on cancel.
func (m *Model) enterFilterForm() {
	m.mode = modeFilter
	m.filtDraft = m.filt
	m.populateFilterInputs()
	m.filterFocus = 0
	for i := range m.filterInputs {
		m.filterInputs[i].Blur()
	}
	m.filterInputs[0].Focus()
}

// populateFilterInputs fills the form inputs from the active filter.
func (m *Model) populateFilterInputs() {
	set := func(i int, v string) { m.filterInputs[i].SetValue(v) }
	set(0, filter.PortRangeString(m.filt.Ports))
	if len(m.filt.Protocols) > 0 {
		names := make([]string, len(m.filt.Protocols))
		for i, p := range m.filt.Protocols {
			names[i] = string(p)
		}
		set(1, strings.Join(names, ","))
	}
	if len(m.filt.States) > 0 {
		names := make([]string, len(m.filt.States))
		for i, s := range m.filt.States {
			names[i] = s.String()
		}
		set(2, strings.Join(names, ","))
	}
	set(3, m.filt.Process)
	if m.filt.PID != 0 {
		set(4, strconv.Itoa(m.filt.PID))
	}
	set(5, m.filt.User)
	set(6, m.filt.Container)
	if m.filt.LocalCIDR != nil {
		set(7, m.filt.LocalCIDR.String())
	}
	if m.filt.RemoteCIDR != nil {
		set(8, m.filt.RemoteCIDR.String())
	}
}

// updateFilterForm handles the 'f' filter form: Tab cycles fields, Enter
// applies, Esc cancels.
func (m Model) updateFilterForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		f, err := m.buildFilterFromInputs()
		if err != nil {
			m.setStatus("filter error: "+err.Error(), 3*time.Second)
			return m, nil
		}
		m.filt = f
		m.rebuild()
		m.clampCursor()
		m.exitFilterForm()
		m.setStatus("filter applied", 2*time.Second)
		return m, nil
	case tea.KeyEscape:
		m.filt = m.filtDraft
		m.rebuild()
		m.clampCursor()
		m.exitFilterForm()
		m.setStatus("filter cancelled", 2*time.Second)
		return m, nil
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyTab, tea.KeyDown:
		m.filterInputs[m.filterFocus].Blur()
		m.filterFocus = (m.filterFocus + 1) % len(m.filterInputs)
		m.filterInputs[m.filterFocus].Focus()
		return m, textinput.Blink
	case tea.KeyShiftTab, tea.KeyUp:
		m.filterInputs[m.filterFocus].Blur()
		m.filterFocus = (m.filterFocus - 1 + len(m.filterInputs)) % len(m.filterInputs)
		m.filterInputs[m.filterFocus].Focus()
		return m, textinput.Blink
	}
	var cmd tea.Cmd
	m.filterInputs[m.filterFocus], cmd = m.filterInputs[m.filterFocus].Update(msg)
	return m, cmd
}

// exitFilterForm returns to table mode and clears the form inputs.
func (m *Model) exitFilterForm() {
	m.mode = modeTable
	for i := range m.filterInputs {
		m.filterInputs[i].Blur()
		m.filterInputs[i].Reset()
	}
}

// buildFilterFromInputs parses the form inputs into a Filter, preserving the
// free-text search field. Returns an error on the first invalid field.
func (m Model) buildFilterFromInputs() (filter.Filter, error) {
	f := m.filt // preserve Text
	f.Ports, f.Protocols, f.States = nil, nil, nil
	f.Process, f.PID, f.User, f.Container = "", 0, "", ""
	f.LocalCIDR, f.RemoteCIDR = nil, nil

	get := func(i int) string { return strings.TrimSpace(m.filterInputs[i].Value()) }
	if v := get(0); v != "" {
		p, err := filter.ParsePorts(v)
		if err != nil {
			return f, err
		}
		f.Ports = p
	}
	if v := get(1); v != "" {
		p, err := filter.ParseProtocols(v)
		if err != nil {
			return f, err
		}
		f.Protocols = p
	}
	if v := get(2); v != "" {
		s, err := filter.ParseStates(v)
		if err != nil {
			return f, err
		}
		f.States = s
	}
	f.Process = get(3)
	if v := get(4); v != "" {
		pid, err := strconv.Atoi(v)
		if err != nil {
			return f, err
		}
		f.PID = pid
	}
	f.User = get(5)
	f.Container = get(6)
	if v := get(7); v != "" {
		c, err := filter.ParseCIDR(v)
		if err != nil {
			return f, err
		}
		f.LocalCIDR = c
	}
	if v := get(8); v != "" {
		c, err := filter.ParseCIDR(v)
		if err != nil {
			return f, err
		}
		f.RemoteCIDR = c
	}
	return f, nil
}

// keyMatches reports whether the key event matches any of the bindings.
func keyMatches(msg tea.KeyMsg, bindings ...key.Binding) bool {
	for _, b := range bindings {
		if b.Enabled() && key.Matches(msg, b) {
			return true
		}
	}
	return false
}
