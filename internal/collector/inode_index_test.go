package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSocketInode(t *testing.T) {
	cases := []struct {
		in   string
		ok   bool
		want uint64
	}{
		{"socket:[12345]", true, 12345},
		{"socket:[0]", true, 0},
		{"socket:[abc]", false, 0},        // non-numeric
		{"pipe:[123]", false, 0},          // not a socket
		{"socket:[12", false, 0},          // missing closing bracket
		{"socket:[]", false, 0},           // empty
		{"anon_inode:[eventpoll]", false, 0},
	}
	for _, c := range cases {
		got, ok := parseSocketInode(c.in)
		assert.Equal(t, c.ok, ok, c.in)
		if ok {
			assert.Equal(t, c.want, got, c.in)
		}
	}
}

func TestBuildInodePIDMap(t *testing.T) {
	fs := newFakeFS()
	// Two PIDs with FDs. PID 1234 owns socket inode 29170 and 9580.
	fs.addLink("/proc/1234/fd/0", "socket:[29170]")
	fs.addLink("/proc/1234/fd/1", "socket:[9580]")
	fs.addLink("/proc/1234/fd/2", "pipe:[555]")
	fs.addLink("/proc/1234/fd/3", "anon_inode:[eventpoll]")
	// PID 5678 owns socket inode 11111.
	fs.addLink("/proc/5678/fd/0", "socket:[11111]")
	// A non-numeric /proc entry that should be ignored.
	fs.addLink("/proc/self/fd/0", "socket:[777]")

	m, err := BuildInodePIDMap(fs, "/proc")
	require.NoError(t, err)
	assert.Equal(t, 1234, m[29170])
	assert.Equal(t, 1234, m[9580])
	assert.Equal(t, 5678, m[11111])
	_, ok := m[777]
	assert.False(t, ok, "/proc/self should be skipped")
}

func TestBuildInodePIDMap_Empty(t *testing.T) {
	fs := newFakeFS()
	m, err := BuildInodePIDMap(fs, "/proc")
	require.NoError(t, err)
	assert.Empty(t, m)
}

func TestIsPermission(t *testing.T) {
	assert.False(t, isPermission(nil))
	assert.True(t, isPermission(&fakeErr{msg: "permission denied"}))
	assert.False(t, isPermission(&fakeErr{msg: "no such file"}))
}
