package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pmtop/pmtop/pkg/netstat"
)

// buildCollectFixture creates a fake /proc with two TCP sockets (one in a
// docker container, one host sshd), a unix socket without an owner, and a
// socket owned by another user (for restricted-mode testing).
func buildCollectFixture() *fakeFS {
	fs := newFakeFS()

	// /proc/net/tcp: 4 sockets with inodes 29170, 9580, 8412, 11111.
	fs.addFile("/proc/net/tcp", tcpFixture)
	// /proc/net/unix: one ownerless socket.
	fs.addFile("/proc/net/unix", "Num RefCount Protocol Flags Type St Inode Path\n0000: 1 0 0 0001 01 999 /tmp/sock\n")

	// PID 1234 = nginx (uid 33) inside docker (cgroup v2 docker scope).
	fs.addLink("/proc/1234/fd/0", "socket:[29170]")
	fs.addFile("/proc/1234/stat", "1234 (nginx) S 1 1234 1234 0 -1 4194560 0 0 0 0 5 5 0 0 20 0 1 0 100 9999 500 0 0 0 0 0 0 0 0 0 0 0 0 0\n")
	fs.addFile("/proc/1234/status", "Name:\tnginx\nUid:\t33\t33\t33\t33\nGid:\t33\t33\t33\t33\nVmRSS:\t  50000 kB\n")
	fs.addFile("/proc/1234/cmdline", "/usr/sbin/nginx\x00-g\x00daemon off;\x00")
	fs.addFile("/proc/1234/cgroup", "0::/system.slice/docker-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.scope\n")

	// PID 5678 = sshd (uid 0) on host.
	fs.addLink("/proc/5678/fd/0", "socket:[8412]")
	fs.addFile("/proc/5678/stat", "5678 (sshd) S 1 5678 5678 0 -1 4194560 0 0 0 0 1 1 0 0 20 0 1 0 50 4096 100 0 0 0 0 0 0 0 0 0 0 0 0 0\n")
	fs.addFile("/proc/5678/status", "Name:\tsshd\nUid:\t0\t0\t0\t0\nGid:\t0\t0\t0\t0\nVmRSS:\t  10000 kB\n")
	fs.addFile("/proc/5678/cmdline", "/usr/sbin/sshd\x00-D\x00")
	fs.addFile("/proc/5678/cgroup", "0::/system.slice/sshd.service\n")

	// PID 9999 = another user's process (uid 1000) owning socket 11111.
	fs.addLink("/proc/9999/fd/0", "socket:[11111]")
	fs.addFile("/proc/9999/stat", "9999 (myapp) S 1 9999 9999 0 -1 0 0 0 0 0 1 1 0 0 20 0 1 0 200 2048 80 0 0 0 0 0 0 0 0 0 0 0 0 0\n")
	fs.addFile("/proc/9999/status", "Name:\tmyapp\nUid:\t1000\t1000\t1000\t1000\nGid:\t1000\t1000\t1000\t1000\nVmRSS:\t  8000 kB\n")
	fs.addFile("/proc/9999/cmdline", "/home/user/myapp\x00")
	fs.addFile("/proc/9999/cgroup", "0::/user.slice/user-1000.slice\n")

	// Shared /proc/stat and /etc files.
	fs.addFile("/proc/stat", "cpu 0\nbtime 1781656942\n")
	fs.addFile("/etc/passwd", "root:x:0:0:root:/root:/bin/bash\nwww-data:x:33:33:www-data:/var/www:/usr/sbin/nologin\nuser:x:1000:1000:User:/home/user:/bin/bash\n")
	fs.addFile("/etc/group", "root:x:0:\nwww-data:x:33:\nuser:x:1000:\n")

	return fs
}

func TestCollect_Enrichment(t *testing.T) {
	fs := buildCollectFixture()
	c := New(fs, "/proc")
	socks, err := c.Collect()
	require.NoError(t, err)
	// 4 TCP + 1 unix = 5
	require.Len(t, socks, 5)

	byInode := map[uint64]netstat.SocketInfo{}
	for _, s := range socks {
		byInode[s.Inode] = s
	}

	// nginx in docker.
	s := byInode[29170]
	assert.Equal(t, 1234, s.PID)
	assert.Equal(t, "nginx", s.ProcessName)
	assert.Equal(t, "www-data", s.User)
	assert.Equal(t, "docker", s.Runtime)
	assert.Equal(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", s.ContainerID)

	// sshd on host.
	s = byInode[8412]
	assert.Equal(t, 5678, s.PID)
	assert.Equal(t, "sshd", s.ProcessName)
	assert.Equal(t, "root", s.User)
	assert.Empty(t, s.Runtime)
	assert.Empty(t, s.ContainerID)

	// myapp owned by user 1000.
	s = byInode[11111]
	assert.Equal(t, 9999, s.PID)
	assert.Equal(t, "myapp", s.ProcessName)
	assert.Equal(t, "user", s.User)

	// Unix socket with no fd owner -> PID 0, no process name.
	s = byInode[999]
	assert.Equal(t, 0, s.PID)
	assert.Empty(t, s.ProcessName)
	assert.Equal(t, "/tmp/sock", s.Path)
}

func TestCollect_Restricted(t *testing.T) {
	fs := buildCollectFixture()
	// Restricted mode as user 1000: only PID 9999's ownership is revealed.
	c := New(fs, "/proc", WithRestricted(1000))
	socks, err := c.Collect()
	require.NoError(t, err)

	byInode := map[uint64]netstat.SocketInfo{}
	for _, s := range socks {
		byInode[s.Inode] = s
	}

	// nginx (uid 33) ownership hidden in restricted mode as user 1000.
	s := byInode[29170]
	assert.Equal(t, 0, s.PID, "other user's PID hidden")
	assert.Empty(t, s.ProcessName)
	assert.Empty(t, s.Runtime)

	// sshd (uid 0) ownership hidden too.
	s = byInode[8412]
	assert.Equal(t, 0, s.PID)
	assert.Empty(t, s.ProcessName)

	// myapp (uid 1000) ownership visible to itself.
	s = byInode[11111]
	assert.Equal(t, 9999, s.PID)
	assert.Equal(t, "myapp", s.ProcessName)
}

func TestCollect_EmptyProc(t *testing.T) {
	fs := newFakeFS()
	// No /proc/net files at all -> Collect returns an error.
	_, err := New(fs, "/proc").Collect()
	assert.Error(t, err)
}

func TestCollect_ProcessDetail(t *testing.T) {
	fs := buildCollectFixture()
	c := New(fs, "/proc")
	pi, err := c.ProcessDetail(1234)
	require.NoError(t, err)
	assert.Equal(t, "nginx", pi.Name)
	assert.Equal(t, `/usr/sbin/nginx -g "daemon off;"`, pi.Cmdline)

	cg, err := c.CgroupDetail(1234)
	require.NoError(t, err)
	assert.Equal(t, "docker", cg.Runtime)
}

func TestJoinPath(t *testing.T) {
	assert.Equal(t, "/proc/net/tcp", joinPath("/proc", "net", "tcp"))
	assert.Equal(t, "/proc/123", joinPath("/proc", "123"))
	assert.Equal(t, "/proc", joinPath("/proc"))
	assert.Equal(t, "net/tcp", joinPath("", "net", "tcp"))
	// No double slash when root already ends with '/'.
	assert.Equal(t, "/proc/1/fd", joinPath("/proc/", "1", "fd"))
}
