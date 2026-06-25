//go:build linux

// Package platform provides platform-specific helpers. The Linux build exposes
// the real effective UID; other platforms return a sentinel.
package platform

import "os"

// CurrentUID returns the effective UID of the calling process.
func CurrentUID() int { return os.Geteuid() }

// IsLinux reports true on Linux builds.
func IsLinux() bool { return true }
