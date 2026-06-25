package collector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const statFixture = "1 (systemd) S 0 1 1 0 -1 4194560 329274 47370258 671 22021 962 1012 41562 79281 20 0 1 0 23 23019520 2381 18446744073709551615 1 1 0 0 0 0 671173123 4096 1260 0 0 0 17 2 0 0 0 0 0 0 0 0 0 0 0 0 0\n"

const statusFixture = `Name:	systemd
Umask:	0000
State:	S (sleeping)
Tgid:	1
Pid:	1
PPid:	0
Uid:	0	0	0	0
Gid:	0	0	0	0
VmRSS:	    2381 kB
VmSize:	   23019520 kB
`

func TestParseStatBytes(t *testing.T) {
	s, err := ParseStatBytes([]byte(statFixture))
	require.NoError(t, err)
	assert.Equal(t, 1, s.PID)
	assert.Equal(t, "systemd", s.Comm)
	assert.Equal(t, "S", s.State)
	assert.Equal(t, 0, s.PPID)
	assert.Equal(t, uint64(962), s.UTime)
	assert.Equal(t, uint64(1012), s.STime)
	assert.Equal(t, uint64(23), s.Starttime)
	assert.Equal(t, uint64(23019520), s.Vsize)
	assert.Equal(t, int64(2381), s.Rss)
}

func TestParseStatBytes_CommWithParens(t *testing.T) {
	// Comm can contain parentheses; the parser must split on first '(' and
	// last ')'.
	in := "42 (chrome (renderer)) R 1 42 42 0 -1 0 0 0 0 0 5 5 0 0 20 0 1 0 100 9999 500 0 0 0 0 0 0 0 0 0 0 0 0 0\n"
	s, err := ParseStatBytes([]byte(in))
	require.NoError(t, err)
	assert.Equal(t, 42, s.PID)
	assert.Equal(t, "chrome (renderer)", s.Comm)
	assert.Equal(t, "R", s.State)
	assert.Equal(t, 1, s.PPID)
	assert.Equal(t, uint64(100), s.Starttime)
}

func TestParseStatBytes_Bad(t *testing.T) {
	_, err := ParseStatBytes([]byte("no parens here"))
	assert.Error(t, err)
	_, err = ParseStatBytes([]byte("abc (comm) S"))
	// "abc" is not a valid pid
	assert.Error(t, err)
}

func TestParseStatusBytes(t *testing.T) {
	st := ParseStatusBytes([]byte(statusFixture))
	assert.Equal(t, "systemd", st.Name)
	assert.Equal(t, uint32(0), st.UID)
	assert.Equal(t, uint32(0), st.GID)
	assert.Equal(t, uint64(2381*1024), st.VmRSS)
	assert.Equal(t, uint64(23019520*1024), st.VmSize)
}

func TestParseStatusBytes_NonRootUser(t *testing.T) {
	in := "Name:\tnginx\nUid:\t33\t33\t33\t33\nGid:\t33\t33\t33\t33\nVmRSS:\t  50000 kB\n"
	st := ParseStatusBytes([]byte(in))
	assert.Equal(t, "nginx", st.Name)
	assert.Equal(t, uint32(33), st.UID)
	assert.Equal(t, uint32(33), st.GID)
	assert.Equal(t, uint64(50000*1024), st.VmRSS)
}

func TestParseCmdlineBytes(t *testing.T) {
	// "/sbin/init\0"
	assert.Equal(t, "/sbin/init", ParseCmdlineBytes([]byte("/sbin/init\x00")))
	// Multiple args with a space -> quoted.
	assert.Equal(t, `/usr/bin/nginx -g "daemon off;"`,
		ParseCmdlineBytes([]byte("/usr/bin/nginx\x00-g\x00daemon off;\x00")))
	assert.Equal(t, "", ParseCmdlineBytes(nil))
	assert.Equal(t, "single", ParseCmdlineBytes([]byte("single")))
}

func TestReadProcess(t *testing.T) {
	fs := newFakeFS()
	fs.addFile("/proc/1/stat", statFixture)
	fs.addFile("/proc/1/status", statusFixture)
	fs.addFile("/proc/1/cmdline", "/sbin/init\x00")
	fs.addLink("/proc/1/exe", "/usr/lib/systemd/systemd")
	fs.addLink("/proc/1/cwd", "/")
	fs.addFile("/proc/1/comm", "systemd\n")
	fs.addFile("/proc/stat", "cpu  0\nbtime 1781656942\n")
	fs.addFile("/etc/passwd", "root:x:0:0:root:/root:/bin/bash\n")
	fs.addFile("/etc/group", "root:x:0:\n")

	pi, err := ReadProcess(fs, "/proc", 1)
	require.NoError(t, err)
	assert.Equal(t, 1, pi.PID)
	assert.Equal(t, "systemd", pi.Name)
	assert.Equal(t, "/sbin/init", pi.Cmdline)
	assert.Equal(t, "/usr/lib/systemd/systemd", pi.Exe)
	assert.Equal(t, "/", pi.CWD)
	assert.Equal(t, uint32(0), pi.UID)
	assert.Equal(t, "root", pi.User)
	assert.Equal(t, "root", pi.Group)
	assert.Equal(t, uint64(2381*1024), pi.VmRSS)
	// starttime ticks=23, HZ=100, btime=1781656942 -> 1781656942 + 0 = ...
	assert.Equal(t, time.Unix(1781656942+int64(23)/defaultHZ, 0).UTC(), pi.StartTime)
}

func TestReadProcess_MissingOptional(t *testing.T) {
	fs := newFakeFS()
	// Only stat present; exe/cwd/status/cmdline missing -> no error, empty fields.
	fs.addFile("/proc/2/stat", "2 (kthreadd) S 0 2 2 0 -1 0 0 0 0 0 0 0 0 0 20 0 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0\n")
	pi, err := ReadProcess(fs, "/proc", 2)
	require.NoError(t, err)
	assert.Equal(t, "kthreadd", pi.Name)
	assert.Empty(t, pi.Exe)
	assert.Empty(t, pi.CWD)
	assert.Empty(t, pi.User)
	assert.True(t, pi.StartTime.IsZero(), "no btime/starttime -> zero start time")
}

func TestBootTime(t *testing.T) {
	fs := newFakeFS()
	fs.addFile("/proc/stat", "cpu  1 2 3\nbtime 1781656942\ncpu0 1\n")
	bt, err := BootTime(fs, "/proc")
	require.NoError(t, err)
	assert.Equal(t, int64(1781656942), bt)
}

func TestBootTime_Missing(t *testing.T) {
	fs := newFakeFS()
	fs.addFile("/proc/stat", "cpu 1 2 3\n")
	_, err := BootTime(fs, "/proc")
	assert.Error(t, err)
}

func TestLookupUserName(t *testing.T) {
	fs := newFakeFS()
	fs.addFile("/etc/passwd", "root:x:0:0:root:/root:/bin/bash\ndaemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin\n")
	name, err := LookupUserName(fs, 0)
	require.NoError(t, err)
	assert.Equal(t, "root", name)
	name, err = LookupUserName(fs, 1)
	require.NoError(t, err)
	assert.Equal(t, "daemon", name)
	_, err = LookupUserName(fs, 999)
	assert.Error(t, err)
}

func TestLookupGroupName(t *testing.T) {
	fs := newFakeFS()
	fs.addFile("/etc/group", "root:x:0:\nwheel:x:10:\n")
	name, err := LookupGroupName(fs, 10)
	require.NoError(t, err)
	assert.Equal(t, "wheel", name)
}

func TestParseKB(t *testing.T) {
	assert.Equal(t, uint64(0), parseKB(""))
	assert.Equal(t, uint64(50000*1024), parseKB("50000 kB"))
}

func TestFirstUint32(t *testing.T) {
	assert.Equal(t, uint32(33), firstUint32("33\t33\t33\t33"))
	assert.Equal(t, uint32(0), firstUint32("nope"))
}
