package collector

import (
	"errors"
	"strconv"

	"github.com/pmtop/pmtop/pkg/netstat"
)

// Collector coordinates a single snapshot of all sockets enriched with their
// owning process and container metadata. It caches per-PID lookups within a
// Collect call so that many sockets sharing a PID incur one /proc read.
type Collector struct {
	fs        FS
	procRoot  string
	restricted bool
	ownerUID  uint32

	// per-Collect caches
	inodeMap    map[uint64]int
	procCache   map[int]*ProcessInfo
	cgroupCache map[int]*CgroupInfo
}

// Option configures a Collector.
type Option func(*Collector)

// WithRestricted puts the collector in restricted mode: ownership (PID,
// process name, container) is only revealed for processes owned by ownerUID;
// other users' sockets keep their socket-level info but hide ownership
// (PRD FR-07-03).
func WithRestricted(ownerUID uint32) Option {
	return func(c *Collector) {
		c.restricted = true
		c.ownerUID = ownerUID
	}
}

// New returns a Collector backed by fs reading procRoot (typically /proc).
func New(fs FS, procRoot string, opts ...Option) *Collector {
	c := &Collector{fs: fs, procRoot: procRoot}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Collect performs a full snapshot: parse every /proc/net/* file, build the
// inode→PID index once, and enrich each socket with process and container
// metadata. Errors from individual files are tolerated (e.g. a missing raw
// file on some kernels) so the snapshot is best-effort.
func (c *Collector) Collect() ([]netstat.SocketInfo, error) {
	c.inodeMap = make(map[uint64]int)
	c.procCache = make(map[int]*ProcessInfo)
	c.cgroupCache = make(map[int]*CgroupInfo)

	socks, err := c.readAllSockets()
	if err != nil {
		return nil, err
	}

	inodes, err := BuildInodePIDMap(c.fs, c.procRoot)
	if err != nil {
		// Proceed without PID mapping rather than failing the whole snapshot.
		inodes = map[uint64]int{}
	}
	c.inodeMap = inodes

	for i := range socks {
		c.enrich(&socks[i])
	}
	return socks, nil
}

// readAllSockets parses every supported /proc/net file. Missing files are
// skipped; only a total failure (e.g. procfs unmounted) is reported.
func (c *Collector) readAllSockets() ([]netstat.SocketInfo, error) {
	var all []netstat.SocketInfo
	var lastErr error

	inetFiles := []struct {
		path  string
		proto netstat.Protocol
	}{
		{"net/tcp", netstat.ProtocolTCP},
		{"net/tcp6", netstat.ProtocolTCP6},
		{"net/udp", netstat.ProtocolUDP},
		{"net/udp6", netstat.ProtocolUDP6},
		{"net/raw", netstat.ProtocolRaw},
		{"net/raw6", netstat.ProtocolRaw6},
	}
	for _, f := range inetFiles {
		s, err := ParseInet(c.fs, joinPath(c.procRoot, f.path), f.proto)
		if err != nil {
			lastErr = err
			continue
		}
		all = append(all, s...)
	}
	if s, err := ParseUnix(c.fs, joinPath(c.procRoot, "net/unix")); err == nil {
		all = append(all, s...)
	} else {
		lastErr = err
	}
	if len(all) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return all, nil
}

// enrich fills the ownership fields of a single socket from the inode map and
// per-PID /proc reads.
func (c *Collector) enrich(s *netstat.SocketInfo) {
	if s.Inode == 0 {
		return
	}
	pid, ok := c.inodeMap[s.Inode]
	if !ok {
		return
	}
	pi, err := c.process(pid)
	if err != nil || pi == nil {
		return
	}
	// Restricted mode: hide ownership for processes not owned by the user.
	if c.restricted && pi.UID != c.ownerUID {
		return
	}
	s.PID = pid
	s.ProcessName = pi.Name
	s.User = pi.User
	s.UID = pi.UID

	if cg := c.cgroup(pid); cg != nil {
		s.Runtime = cg.Runtime
		s.ContainerID = cg.ContainerID
	}
}

// process returns cached ProcessInfo for pid.
func (c *Collector) process(pid int) (*ProcessInfo, error) {
	if pi, ok := c.procCache[pid]; ok {
		return pi, nil
	}
	pi, err := ReadProcess(c.fs, c.procRoot, pid)
	if err != nil {
		return nil, err
	}
	c.procCache[pid] = &pi
	return &pi, nil
}

// cgroup returns cached CgroupInfo for pid.
func (c *Collector) cgroup(pid int) *CgroupInfo {
	if cg, ok := c.cgroupCache[pid]; ok {
		return cg
	}
	data, err := c.fs.ReadFile(joinPath(c.procRoot, strconv.Itoa(pid), "cgroup"))
	if err != nil {
		return nil
	}
	cg := ParseCgroupBytes(data)
	c.cgroupCache[pid] = &cg
	return &cg
}

// ProcessDetail returns full ProcessInfo for a PID (uncached, fresh read),
// used by the process detail panel (PRD FR-04-02).
func (c *Collector) ProcessDetail(pid int) (ProcessInfo, error) {
	return ReadProcess(c.fs, c.procRoot, pid)
}

// CgroupDetail returns fresh CgroupInfo for a PID.
func (c *Collector) CgroupDetail(pid int) (CgroupInfo, error) {
	data, err := c.fs.ReadFile(joinPath(c.procRoot, strconv.Itoa(pid), "cgroup"))
	if err != nil {
		return CgroupInfo{}, err
	}
	return ParseCgroupBytes(data), nil
}

// ErrNoProc is returned when procfs appears unavailable.
var ErrNoProc = errors.New("procfs unavailable")
