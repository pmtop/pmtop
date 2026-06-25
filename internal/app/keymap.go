// Package app implements the interactive Bubble Tea TUI for pmtop.
//
// The TUI is decoupled from /proc via the DataSource interface so it can be
// unit-tested on any platform with a fake data source.
package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the default key bindings (PRD 6.4). All bindings are
// configurable via the config file (FR-10-03, wired in M5).
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	PageUp  key.Binding
	PageDn  key.Binding
	Home    key.Binding
	End     key.Binding
	Enter   key.Binding
	Escape  key.Binding
	Search  key.Binding
	Filter  key.Binding
	Sort    key.Binding
	SortDir key.Binding
	Kill    key.Binding
	Pause   key.Binding
	Refresh key.Binding
	Export  key.Binding
	Help    key.Binding
	Quit    key.Binding
}

// DefaultKeyMap returns the standard set of key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		PageUp:  key.NewBinding(key.WithKeys("pgup"), key.WithHelp("PgUp", "page up")),
		PageDn:  key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("PgDn", "page down")),
		Home:    key.NewBinding(key.WithKeys("home"), key.WithHelp("Home", "top")),
		End:     key.NewBinding(key.WithKeys("end"), key.WithHelp("End", "bottom")),
		Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "detail")),
		Escape:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "close")),
		Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Filter:  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter")),
		Sort:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort col")),
		SortDir: key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sort dir")),
		Kill:    key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "signal")),
		Pause:   key.NewBinding(key.WithKeys(" "), key.WithHelp("Space", "pause")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Export:  key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export")),
		Help:    key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "help")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// ShortHelp returns a compact, ordered list of help entries for the bottom bar.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Sort, k.Refresh, k.Pause, k.Help, k.Quit}
}

// FullHelp returns grouped help entries for the F1 help overlay.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDn, k.Home, k.End},
		{k.Enter, k.Search, k.Filter, k.Sort, k.SortDir, k.Kill},
		{k.Pause, k.Refresh, k.Export, k.Help, k.Quit},
	}
}
