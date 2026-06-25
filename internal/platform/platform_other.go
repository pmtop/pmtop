//go:build !linux

// Package platform provides platform-specific helpers. Non-Linux builds return
// sentinel values so the project cross-compiles for development/test setups.
package platform

// CurrentUID returns -1 on non-Linux platforms (no /proc, no UIDs).
func CurrentUID() int { return -1 }

// IsLinux reports false on non-Linux builds.
func IsLinux() bool { return false }
