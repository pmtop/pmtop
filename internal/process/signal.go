// Package process implements process management: signal sending (FR-06),
// process detail collection (FR-04), and owning-package lookup (FR-04-05).
package process

import (
	"errors"
	"strconv"
	"strings"
)

// Signal is a POSIX signal that pmtop can send to a process.
type Signal struct {
	Name string
	Num  int
	Desc string
}

// Signals is the set of signals offered by the signal dialog (FR-06-02).
// SIGTERM is the default selection.
var Signals = []Signal{
	{"SIGHUP", 1, "Reload config"},
	{"SIGINT", 2, "Interrupt"},
	{"SIGTERM", 15, "Graceful stop"},
	{"SIGKILL", 9, "Force kill"},
	{"SIGUSR1", 10, "User-defined 1"},
	{"SIGUSR2", 12, "User-defined 2"},
}

// DefaultSignal returns SIGTERM (FR-06-02).
func DefaultSignal() Signal { return Signals[2] }

// ParseSignal accepts a signal name ("SIGTERM") or number ("15") and returns
// the matching Signal. Numbers are validated against the known signal table.
func ParseSignal(s string) (Signal, bool) {
	s = strings.TrimSpace(strings.ToUpper(s))
	for _, sig := range Signals {
		if sig.Name == s {
			return sig, true
		}
	}
	if n, err := strconv.Atoi(s); err == nil {
		for _, sig := range Signals {
			if sig.Num == n {
				return sig, true
			}
		}
	}
	return Signal{}, false
}

// ValidatePID returns an error if pid is not a positive integer.
func ValidatePID(pid int) error {
	if pid <= 0 {
		return errors.New("invalid pid: must be > 0")
	}
	return nil
}
