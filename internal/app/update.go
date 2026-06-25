package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"

	"github.com/pmtop/pmtop/internal/ui"
)

// Update handles all messages: window resizing, refresh ticks, and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tbl.SetWidth(msg.Width)
		// Reserve 2 lines for top status + bottom hint bars.
		h := msg.Height - 2
		if h < 3 {
			h = 3
		}
		m.tbl.SetHeight(h)
		m.tbl.SetColumns(ui.BuildColumns(msg.Width))
		return m, nil

	case tickMsg:
		// Schedule the next tick regardless of pause state.
		cmds := []tea.Cmd{tickCmd(m.interval)}
		if !m.paused {
			m.refresh()
		}
		return m, tea.Batch(cmds...)

	case refreshMsg:
		m.refresh()
		return m, nil

	case tea.KeyMsg:
		// Nothing to navigate if there are no rows.
		if len(m.socks) == 0 && !keyMatches(msg, m.keys.Quit) {
			return m, nil
		}
		switch {
		case keyMatches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

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

		// Actions wired in later milestones; show a hint for now.
		case keyMatches(msg, m.keys.Enter, m.keys.Search, m.keys.Filter,
			m.keys.Kill, m.keys.Export, m.keys.Help):
			m.setStatus("not available yet", time.Second)
			return m, nil
		}
	}

	return m, nil
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
