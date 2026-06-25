// Package elevate handles privilege detection and the opt-in sudo re-launch
// flow (PRD FR-07). Elevation is never automatic: the user sees a banner and
// explicitly chooses to re-launch with sudo.
package elevate

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// State describes the current privilege context.
type State struct {
	IsRoot       bool   // effective UID == 0
	UID          int    // effective UID
	NoElevate    bool   // --no-elevate flag set (CI/automation)
	Restricted   bool   // non-root without elevation
	SudoAvail    bool   // sudo appears to be on PATH
	BinaryPath   string // path to the current executable, for re-exec
}

// Detect determines the privilege state. noElevate mirrors the --no-elevate
// flag (FR-07-04). On non-Linux platforms a sentinel state is returned.
func Detect(noElevate bool) State {
	st := State{NoElevate: noElevate}
	st.UID = effectiveUID()
	st.IsRoot = st.UID == 0
	st.Restricted = !st.IsRoot && !noElevate
	st.SudoAvail = commandExists("sudo")
	if exe, err := os.Executable(); err == nil {
		st.BinaryPath = exe
	}
	return st
}

// IsLinux reports whether the current build targets Linux.
func IsLinux() bool { return isLinux }

// BannerText returns the non-root restricted-mode banner (PRD 6.5).
func BannerText() string {
	return "⚠ Running without root. Only your own processes are shown.\n" +
		"  Run `sudo pmtop` for full port and process visibility.\n" +
		"  Press S to restart with sudo, or any key to continue."
}

// ErrNoSudo is returned when sudo is not available.
var ErrNoSudo = errors.New("sudo not found on PATH")

// ErrNoBinary is returned when the current executable path is unknown.
var ErrNoBinary = errors.New("cannot determine own executable path")

// RelaunchArgs returns the argv to re-exec the current binary under sudo,
// preserving all arguments. The returned slice starts with "sudo".
func RelaunchArgs() ([]string, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, ErrNoBinary
	}
	args := append([]string{"sudo", "--preserve-env", exe}, os.Args[1:]...)
	return args, nil
}

// Relaunch re-execs the current binary under sudo, replacing this process.
// Returns ErrNoSudo if sudo is missing. On success it never returns (the
// process is replaced); on exec failure the error is returned.
func Relaunch() error {
	if !commandExists("sudo") {
		return ErrNoSudo
	}
	args, err := RelaunchArgs()
	if err != nil {
		return err
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo re-exec failed: %w", err)
	}
	os.Exit(exitCodeFrom(err))
	return nil
}

// exitCodeFrom extracts the exit code from an exec error, defaulting to 1.
func exitCodeFrom(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return 1
}

// commandExists reports whether name is found on PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// effectiveUID returns the effective UID. On Linux this is os.Geteuid; on
// other platforms it returns -1 (no /proc). Defined per-OS below.
func effectiveUID() int { return currentEUID() }

// currentEUID is platform-specific; see elevate_linux.go / elevate_other.go.
var currentEUID = func() int { return -1 }

// ParseUID is a small helper for tests that parse a numeric uid string.
func ParseUID(s string) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, err
	}
	return v, nil
}
