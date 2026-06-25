//go:build !linux

package elevate

const isLinux = false

// On non-Linux platforms there is no /proc and no real UID; -1 signals
// "unsupported" so the TUI refuses to start but the project still compiles.
func init() {
	currentEUID = func() int { return -1 }
}
