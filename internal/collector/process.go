package collector

import (
	"strconv"
	"strings"
	"time"
)

// linuxPageSize is the page size used to interpret /proc/<pid>/stat rss
// (reported in pages). It is 4096 on all supported Linux architectures.
const linuxPageSize = 4096

// defaultHZ is the USER_HZ clock frequency used to convert tick fields
// (utime/stime/starttime) to seconds. USER_HZ is 100 on all mainstream Linux.
const defaultHZ = 100

// ProcessInfo holds the enriched attributes of a single process, gathered from
// several /proc/<pid> files. Used both for socket ownership enrichment and for
// the process detail side panel (PRD FR-04).
type ProcessInfo struct {
	PID        int
	PPID       int
	Name       string // comm (truncated to 15 chars by the kernel)
	Cmdline    string // full command line with arguments
	Exe        string // resolved executable path
	CWD        string // resolved working directory
	State      string // R, S, Z, T, ...
	UID        uint32
	GID        uint32
	User       string
	Group      string
	VmRSS      uint64 // resident set size, bytes
	VmSize     uint64 // virtual size, bytes
	UTime      uint64 // user CPU ticks
	STime      uint64 // kernel CPU ticks
	StartTime  time.Time
	StartTicks uint64 // raw starttime field (ticks since boot)
}

// ReadProcess assembles a ProcessInfo for pid by reading the relevant
// /proc files. Missing optional files (exe/cwd for kernels without ptrace
// permission) are left empty rather than causing an error.
func ReadProcess(fs FS, procRoot string, pid int) (ProcessInfo, error) {
	pi := ProcessInfo{PID: pid}

	base := joinPath(procRoot, strconv.Itoa(pid))

	if statData, err := fs.ReadFile(joinPath(base, "stat")); err == nil {
		if s, err := ParseStatBytes(statData); err == nil {
			pi.PID = s.PID
			pi.PPID = s.PPID
			pi.Name = s.Comm
			pi.State = s.State
			pi.UTime = s.UTime
			pi.STime = s.STime
			pi.StartTicks = s.Starttime
			pi.VmSize = s.Vsize
			pi.VmRSS = uint64(s.Rss) * linuxPageSize
		}
	}

	if statusData, err := fs.ReadFile(joinPath(base, "status")); err == nil {
		st := ParseStatusBytes(statusData)
		if st.UID > 0 || pi.UID == 0 {
			pi.UID = st.UID
		}
		if st.GID > 0 || pi.GID == 0 {
			pi.GID = st.GID
		}
		if st.VmRSS > 0 {
			pi.VmRSS = st.VmRSS
		}
		if st.VmSize > 0 {
			pi.VmSize = st.VmSize
		}
		if pi.Name == "" {
			pi.Name = st.Name
		}
	}

	if cmdData, err := fs.ReadFile(joinPath(base, "cmdline")); err == nil {
		pi.Cmdline = ParseCmdlineBytes(cmdData)
	}
	if pi.Name == "" {
		if comm, err := fs.ReadFile(joinPath(base, "comm")); err == nil {
			pi.Name = strings.TrimSpace(string(comm))
		}
	}

	// exe and cwd are symlinks; unreadable (permission denied) yields empty.
	if exe, err := fs.Readlink(joinPath(base, "exe")); err == nil {
		pi.Exe = exe
	}
	if cwd, err := fs.Readlink(joinPath(base, "cwd")); err == nil {
		pi.CWD = cwd
	}

	// Resolve user/group names from /etc/passwd and /etc/group.
	if name, err := LookupUserName(fs, pi.UID); err == nil {
		pi.User = name
	}
	if name, err := LookupGroupName(fs, pi.GID); err == nil {
		pi.Group = name
	}

	// Compute wall-clock start time from boot time + starttime ticks.
	if pi.StartTicks > 0 {
		if boot, err := BootTime(fs, procRoot); err == nil {
			pi.StartTime = time.Unix(boot+int64(pi.StartTicks)/defaultHZ, 0).UTC()
		}
	}

	return pi, nil
}

// StatInfo is the parsed content of /proc/<pid>/stat.
type StatInfo struct {
	PID       int
	Comm      string
	State     string
	PPID      int
	UTime     uint64
	STime     uint64
	Starttime uint64
	Vsize     uint64
	Rss       int64 // pages
}

// ParseStatBytes parses /proc/<pid>/stat content. The comm field is wrapped in
// parentheses and may itself contain parentheses and spaces, so the line is
// split around the first '(' and the last ')'.
func ParseStatBytes(data []byte) (StatInfo, error) {
	s := strings.TrimRight(strings.TrimRight(string(data), "\n"), "\r")
	firstOpen := strings.IndexByte(s, '(')
	lastClose := strings.LastIndexByte(s, ')')
	if firstOpen < 0 || lastClose < 0 || lastClose < firstOpen {
		return StatInfo{}, errParse("stat: missing comm parens")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(s[:firstOpen]))
	if err != nil {
		return StatInfo{}, errParse("stat: bad pid")
	}
	comm := s[firstOpen+1 : lastClose]
	rest := strings.Fields(s[lastClose+1:])
	// rest indices (0-based after ')'):
	// 0 state, 1 ppid, 2 pgrp, 3 session, 4 tty_nr, 5 tpgid, 6 flags,
	// 7 minflt, 8 cminflt, 9 majflt, 10 cmajflt, 11 utime, 12 stime,
	// 13 cutime, 14 cstime, 15 priority, 16 nice, 17 num_threads,
	// 18 itrealvalue, 19 starttime, 20 vsize, 21 rss
	get := func(i int) string {
		if i < len(rest) {
			return rest[i]
		}
		return ""
	}
	st := StatInfo{PID: pid, Comm: comm, State: get(0)}
	st.PPID, _ = strconv.Atoi(get(1))
	st.UTime, _ = strconv.ParseUint(get(11), 10, 64)
	st.STime, _ = strconv.ParseUint(get(12), 10, 64)
	st.Starttime, _ = strconv.ParseUint(get(19), 10, 64)
	st.Vsize, _ = strconv.ParseUint(get(20), 10, 64)
	st.Rss, _ = strconv.ParseInt(get(21), 10, 64)
	return st, nil
}

// StatusInfo is the parsed content of /proc/<pid>/status.
type StatusInfo struct {
	Name   string
	UID    uint32
	GID    uint32
	VmRSS  uint64
	VmSize uint64
}

// ParseStatusBytes parses /proc/<pid>/status content (key: value lines).
func ParseStatusBytes(data []byte) StatusInfo {
	var st StatusInfo
	for _, line := range splitLines(data) {
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colon])
		val := strings.TrimSpace(line[colon+1:])
		switch key {
		case "Name":
			st.Name = val
		case "Uid":
			st.UID = firstUint32(val)
		case "Gid":
			st.GID = firstUint32(val)
		case "VmRSS":
			st.VmRSS = parseKB(val)
		case "VmSize":
			st.VmSize = parseKB(val)
		}
	}
	return st
}

// ParseCmdlineBytes parses /proc/<pid>/cmdline (null-separated argv). Args are
// joined with spaces; a trailing null is stripped.
func ParseCmdlineBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	// Drop a single trailing NUL.
	if data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}
	parts := strings.Split(string(data), "\x00")
	// Quote args containing spaces so the command line is unambiguous.
	for i, p := range parts {
		if strings.ContainsAny(p, " \t") && !strings.HasPrefix(p, "\"") {
			parts[i] = "\"" + p + "\""
		}
	}
	return strings.Join(parts, " ")
}

// BootTime reads /proc/stat and returns the btime field (seconds since epoch).
func BootTime(fs FS, procRoot string) (int64, error) {
	data, err := fs.ReadFile(joinPath(procRoot, "stat"))
	if err != nil {
		return 0, err
	}
	for _, line := range splitLines(data) {
		if strings.HasPrefix(line, "btime ") {
			v, err := strconv.ParseInt(strings.TrimSpace(line[6:]), 10, 64)
			if err != nil {
				return 0, err
			}
			return v, nil
		}
	}
	return 0, errParse("btime not found")
}

// LookupUserName resolves a UID to a username via /etc/passwd.
func LookupUserName(fs FS, uid uint32) (string, error) {
	return lookupIDFile(fs, "/etc/passwd", uid, 2)
}

// LookupGroupName resolves a GID to a group name via /etc/group.
func LookupGroupName(fs FS, gid uint32) (string, error) {
	return lookupIDFile(fs, "/etc/group", gid, 2)
}

// lookupIDFile parses /etc/passwd or /etc/group (colon-separated) and returns
// the name (field 0) matching the given numeric id (field idIdx, typically 2).
func lookupIDFile(fs FS, path string, id uint32, idIdx int) (string, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range splitLines(data) {
		if line == "" {
			continue
		}
		f := strings.Split(line, ":")
		if len(f) <= idIdx {
			continue
		}
		v, err := strconv.ParseUint(f[idIdx], 10, 32)
		if err != nil {
			continue
		}
		if uint32(v) == id {
			return f[0], nil
		}
	}
	return "", errParse("id not found")
}

// firstUint32 returns the first whitespace-separated uint32 in s.
func firstUint32(s string) uint32 {
	for _, f := range strings.Fields(s) {
		if v, err := strconv.ParseUint(f, 10, 32); err == nil {
			return uint32(v)
		}
	}
	return 0
}

// parseKB parses values like "12345 kB" into bytes.
func parseKB(s string) uint64 {
	for _, f := range strings.Fields(s) {
		if v, err := strconv.ParseUint(f, 10, 64); err == nil {
			return v * 1024
		}
	}
	return 0
}

// errParse returns a simple parse error.
type parseError string

func (e parseError) Error() string { return string(e) }

func errParse(s string) error { return parseError(s) }
