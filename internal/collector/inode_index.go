package collector

import (
	"os"
	"strconv"
	"strings"
)

// BuildInodePIDMap performs a single pass over /proc/<pid>/fd/* and returns a
// map from socket inode to owning PID (PRD 5.4). Unreadable entries (e.g.
// permission denied for other users' processes) are skipped silently.
//
// Complexity is O(total FDs) rather than O(sockets × pids × fds) because the
// inode map is built once and reused to enrich every socket.
func BuildInodePIDMap(fs FS, procRoot string) (map[uint64]int, error) {
	index := make(map[uint64]int)

	pidDirs, err := fs.Glob(joinPath(procRoot, "[0-9]*"))
	if err != nil {
		return nil, err
	}
	for _, pidDir := range pidDirs {
		base := pidDir
		if idx := strings.LastIndexByte(base, '/'); idx >= 0 {
			base = base[idx+1:]
		}
		pid, err := strconv.Atoi(base)
		if err != nil || pid <= 0 {
			continue
		}
		fdLinks, err := fs.Glob(joinPath(pidDir, "fd", "*"))
		if err != nil {
			continue
		}
		for _, link := range fdLinks {
			target, err := fs.Readlink(link)
			if err != nil {
				continue // permission denied or stale fd
			}
			if inode, ok := parseSocketInode(target); ok {
				// First writer wins; a socket inode is owned by exactly one PID.
				if _, exists := index[inode]; !exists {
					index[inode] = pid
				}
			}
		}
	}
	return index, nil
}

// parseSocketInode extracts the inode number from a "socket:[<inode>]" symlink
// target such as those found under /proc/<pid>/fd/.
func parseSocketInode(target string) (uint64, bool) {
	const prefix = "socket:["
	if !strings.HasPrefix(target, prefix) || !strings.HasSuffix(target, "]") {
		return 0, false
	}
	inner := target[len(prefix) : len(target)-1]
	inode, err := strconv.ParseUint(inner, 10, 64)
	if err != nil {
		return 0, false
	}
	return inode, true
}

// isPermission reports whether err is a permission-denied error.
func isPermission(err error) bool {
	return err != nil && (os.IsPermission(err) || strings.Contains(err.Error(), "permission denied"))
}
