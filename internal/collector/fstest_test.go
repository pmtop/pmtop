package collector

import (
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"
)

// fakeFS is an in-memory FS for unit-testing the collector on any platform
// without a real Linux /proc. It is defined in a _test.go file so it never
// ships in the production binary.
type fakeFS struct {
	files   map[string][]byte // regular file contents
	links   map[string]string // symlink targets
}

func newFakeFS() *fakeFS {
	return &fakeFS{
		files: make(map[string][]byte),
		links: make(map[string]string),
	}
}

func (f *fakeFS) ReadFile(name string) ([]byte, error) {
	name = cleanPath(name)
	if b, ok := f.files[name]; ok {
		return b, nil
	}
	return nil, &fakeErr{notExist: true, msg: "no such file: " + name}
}

func (f *fakeFS) Readlink(name string) (string, error) {
	name = cleanPath(name)
	if t, ok := f.links[name]; ok {
		return t, nil
	}
	return "", &fakeErr{notExist: true, msg: "no such link: " + name}
}

func (f *fakeFS) Glob(pattern string) ([]string, error) {
	pattern = cleanPath(pattern)
	// Candidates include every file/link key plus all of their ancestor
	// directories, so that patterns like "/proc/[0-9]*" match directory entries
	// even though the fake FS only stores leaf paths (mirroring how a real
	// filesystem enumerates directories).
	candidates := map[string]bool{}
	add := func(p string) { candidates[cleanPath(p)] = true }
	for k := range f.files {
		add(k)
		addAncestors(k, add)
	}
	for k := range f.links {
		add(k)
		addAncestors(k, add)
	}
	var matched []string
	for c := range candidates {
		if ok, _ := path.Match(pattern, c); ok {
			matched = append(matched, c)
		}
	}
	sort.Strings(matched)
	return matched, nil
}

// addAncestors adds every parent directory of p (excluding "/" itself) via add.
func addAncestors(p string, add func(string)) {
	for {
		idx := strings.LastIndexByte(p, '/')
		if idx <= 0 {
			break
		}
		p = p[:idx]
		add(p)
	}
}

func (f *fakeFS) Stat(name string) (fs.FileInfo, error) {
	name = cleanPath(name)
	if b, ok := f.files[name]; ok {
		return &fakeFileInfo{name: name, size: int64(len(b)), mode: 0644}, nil
	}
	if _, ok := f.links[name]; ok {
		return &fakeFileInfo{name: name, mode: 0777 | fs.ModeSymlink}, nil
	}
	return nil, &fakeErr{notExist: true, msg: "no such file: " + name}
}

// addFile inserts a regular file.
func (f *fakeFS) addFile(name, content string) {
	f.files[cleanPath(name)] = []byte(content)
}

// addLink inserts a symlink.
func (f *fakeFS) addLink(name, target string) {
	f.links[cleanPath(name)] = target
}

func cleanPath(p string) string {
	if p == "" {
		return p
	}
	// Normalize backslashes to forward slashes for cross-platform test stability.
	p = strings.ReplaceAll(p, "\\", "/")
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}

// fakeErr is a minimal error implementing IsNotExist for graceful handling.
type fakeErr struct {
	notExist bool
	msg      string
}

func (e *fakeErr) Error() string { return e.msg }

// fakeFileInfo is a minimal FileInfo for the fake FS.
type fakeFileInfo struct {
	name  string
	size  int64
	mode  fs.FileMode
	isDir bool
}

func (fi *fakeFileInfo) Name() string       { return fi.name }
func (fi *fakeFileInfo) Size() int64        { return fi.size }
func (fi *fakeFileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi *fakeFileInfo) ModTime() time.Time { return time.Unix(0, 0) }
func (fi *fakeFileInfo) IsDir() bool        { return fi.isDir }
func (fi *fakeFileInfo) Sys() any           { return nil }
