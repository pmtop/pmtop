package collector

import (
	"strings"
)

// CgroupLine is a single line of /proc/<pid>/cgroup.
type CgroupLine struct {
	HierarchyID string
	Subsystems  string
	Path        string
}

// CgroupInfo is the parsed result of /proc/<pid>/cgroup plus container
// runtime detection (PRD 5.6, FR-05-01).
type CgroupInfo struct {
	Version     int // 1, 2, or 0 if unknown
	Lines       []CgroupLine
	Runtime     string // docker, containerd, podman, crio, or ""
	ContainerID string // hex container id (long or short form)
}

// v1Markers map cgroup v1 path substrings to runtime names.
var v1Markers = []struct {
	runtime string
	marker  string
}{
	{"docker", "/docker/"},
	{"containerd", "/containerd/"},
	{"podman", "/libpod/"},
	{"podman", "/podman/"},
	{"crio", "/crio/"},
	{"crio", "/cri-containerd/"},
}

// v2ScopePrefixes map cgroup v2 scope name prefixes to runtime names.
var v2ScopePrefixes = []struct {
	runtime string
	prefix  string
}{
	{"docker", "docker-"},
	{"containerd", "containerd-"},
	{"podman", "libpod-"},
	{"crio", "crio-"},
}

// ParseCgroupBytes parses /proc/<pid>/cgroup content.
func ParseCgroupBytes(data []byte) CgroupInfo {
	var cg CgroupInfo
	for _, line := range splitLines(data) {
		if line == "" {
			continue
		}
		// Format: "<hierarchy_id>:<subsystems>:<path>"
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		cl := CgroupLine{
			HierarchyID: parts[0],
			Subsystems:  parts[1],
			Path:        parts[2],
		}
		cg.Lines = append(cg.Lines, cl)
		if cl.HierarchyID == "0" && cl.Subsystems == "" {
			cg.Version = 2
		} else if cg.Version == 0 {
			cg.Version = 1
		}
	}
	cg.Runtime, cg.ContainerID = detectContainer(cg.Lines, cg.Version)
	return cg
}

// detectContainer scans cgroup lines and returns the container runtime name and
// container id, if any. Returns ("", "") for host processes.
func detectContainer(lines []CgroupLine, version int) (string, string) {
	// Prefer v2 scope patterns when cgroup v2 is in use.
	if version == 2 {
		for _, l := range lines {
			if rt, id, ok := matchV2Scope(l.Path); ok {
				return rt, id
			}
		}
	}
	// v1 markers (also serve as fallback for v2 unmatched paths).
	for _, l := range lines {
		if rt, id, ok := matchV1Marker(l.Path); ok {
			return rt, id
		}
	}
	// Last resort: try v2 scope patterns even on v1 (mixed setups).
	for _, l := range lines {
		if rt, id, ok := matchV2Scope(l.Path); ok {
			return rt, id
		}
	}
	return "", ""
}

// matchV1Marker looks for a "/<runtime>/<id>" substring in a v1 cgroup path.
func matchV1Marker(path string) (string, string, bool) {
	for _, m := range v1Markers {
		idx := strings.Index(path, m.marker)
		if idx < 0 {
			continue
		}
		rest := path[idx+len(m.marker):]
		id := extractHexSegment(rest)
		if id != "" {
			return m.runtime, id, true
		}
	}
	return "", "", false
}

// matchV2Scope looks for a "<runtime>-<id>.scope" segment in a v2 cgroup path.
func matchV2Scope(path string) (string, string, bool) {
	for _, p := range v2ScopePrefixes {
		idx := strings.Index(path, p.prefix)
		if idx < 0 {
			continue
		}
		rest := path[idx+len(p.prefix):]
		// Strip a trailing ".scope" (and anything after it).
		if dot := strings.IndexByte(rest, '.'); dot >= 0 {
			rest = rest[:dot]
		}
		id := extractHexSegment(rest)
		if id != "" {
			return p.runtime, id, true
		}
	}
	return "", "", false
}

// extractHexSegment returns the leading run of hex characters of length >= 12
// (Docker/Podman container IDs are 64 hex chars; short IDs are 12). Non-hex
// characters terminate the run.
func extractHexSegment(s string) string {
	// Trim leading path separators.
	s = strings.TrimLeft(s, "/")
	end := 0
	for end < len(s) && isHexByte(s[end]) {
		end++
	}
	id := s[:end]
	if len(id) < 12 {
		return ""
	}
	return id
}

// isHexByte reports whether b is an ASCII hex digit.
func isHexByte(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}
