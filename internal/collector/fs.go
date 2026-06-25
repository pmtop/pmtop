// Package collector implements the data collection engine that reads Linux
// /proc and (optionally) container runtime APIs to build a unified view of
// sockets, processes, and containers.
//
// All collection functions are parameterized by an FS interface and a root
// path (typically "/proc") so they can be unit-tested with in-memory fixtures
// on any platform, without a real Linux /proc.
package collector

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FS is the minimal filesystem interface the collector needs to read /proc.
// It is satisfied by osFS (the real OS) and by test doubles.
type FS interface {
	// ReadFile reads the entire file at name.
	ReadFile(name string) ([]byte, error)
	// Readlink returns the destination of the symbolic link at name.
	Readlink(name string) (string, error)
	// Glob returns the paths matching the given shell pattern.
	Glob(pattern string) ([]string, error)
	// Stat returns FileInfo for name.
	Stat(name string) (fs.FileInfo, error)
}

// osFS is the production FS backed by the host operating system.
type osFS struct{}

// NewOSFS returns an FS that reads from the real filesystem.
func NewOSFS() FS { return osFS{} }

func (osFS) ReadFile(name string) ([]byte, error)        { return os.ReadFile(name) }
func (osFS) Readlink(name string) (string, error)        { return os.Readlink(name) }
func (osFS) Glob(pattern string) ([]string, error)       { return filepath.Glob(pattern) }
func (osFS) Stat(name string) (fs.FileInfo, error)       { return os.Stat(name) }

// DefaultProcRoot is the standard location of procfs on Linux.
const DefaultProcRoot = "/proc"

// joinPath joins a root and relative path elements using forward slashes,
// independent of the host OS path separator, so /proc paths are correct even
// when the collector is compiled/run on Windows for cross-compilation tests.
func joinPath(root string, elems ...string) string {
	out := root
	for _, e := range elems {
		if e == "" {
			continue
		}
		if out == "" {
			out = e
			continue
		}
		if len(out) > 0 && out[len(out)-1] == '/' {
			out += e
		} else {
			out += "/" + e
		}
	}
	return out
}
