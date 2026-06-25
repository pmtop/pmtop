package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCgroupBytes_V2Host(t *testing.T) {
	cg := ParseCgroupBytes([]byte("0::/init.scope\n"))
	assert.Equal(t, 2, cg.Version)
	assert.Empty(t, cg.Runtime)
	assert.Empty(t, cg.ContainerID)
}

func TestParseCgroupBytes_V2Docker(t *testing.T) {
	// Docker container under systemd cgroup v2.
	in := "0::/system.slice/docker-abc123def4567890123456789012345678901234567890123456789012345678.scope\n"
	cg := ParseCgroupBytes([]byte(in))
	assert.Equal(t, 2, cg.Version)
	assert.Equal(t, "docker", cg.Runtime)
	assert.Equal(t, "abc123def4567890123456789012345678901234567890123456789012345678", cg.ContainerID)
}

func TestParseCgroupBytes_V2Containerd(t *testing.T) {
	in := "0::/system.slice/containerd-abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890.scope\n"
	cg := ParseCgroupBytes([]byte(in))
	assert.Equal(t, "containerd", cg.Runtime)
	assert.Len(t, cg.ContainerID, 64)
}

func TestParseCgroupBytes_V2Podman(t *testing.T) {
	in := "0::/machine.slice/libpod-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.scope\n"
	cg := ParseCgroupBytes([]byte(in))
	assert.Equal(t, "podman", cg.Runtime)
	assert.Len(t, cg.ContainerID, 64)
}

func TestParseCgroupBytes_V1Docker(t *testing.T) {
	in := "11:cpuset:/docker/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n" +
		"10:memory:/docker/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n"
	cg := ParseCgroupBytes([]byte(in))
	assert.Equal(t, 1, cg.Version)
	assert.Equal(t, "docker", cg.Runtime)
	assert.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", cg.ContainerID)
}

func TestParseCgroupBytes_V1PodmanLibpod(t *testing.T) {
	in := "11:cpuset:/libpod/abcdef123456abcdef123456abcdef123456abcdef123456abcdef123456abcd\n"
	cg := ParseCgroupBytes([]byte(in))
	assert.Equal(t, "podman", cg.Runtime)
	assert.Len(t, cg.ContainerID, 64)
}

func TestParseCgroupBytes_HostV1(t *testing.T) {
	in := "11:cpuset:/\n10:memory:/user.slice\n"
	cg := ParseCgroupBytes([]byte(in))
	assert.Equal(t, 1, cg.Version)
	assert.Empty(t, cg.Runtime)
	assert.Empty(t, cg.ContainerID)
}

func TestParseCgroupBytes_ShortIDRejected(t *testing.T) {
	// IDs must be >= 12 hex chars.
	in := "0::/system.slice/docker-ab12.scope\n"
	cg := ParseCgroupBytes([]byte(in))
	assert.Empty(t, cg.Runtime, "short id (<12) should not match")
	assert.Empty(t, cg.ContainerID)
}

func TestParseCgroupBytes_MultipleLinesFirstWins(t *testing.T) {
	// Mixed v1+v2 lines; the v2 scope pattern is preferred and matches line 2.
	in := "11:cpuset:/docker/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n" +
		"0::/system.slice/docker-bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.scope\n"
	cg := ParseCgroupBytes([]byte(in))
	assert.Equal(t, 2, cg.Version)
	assert.Equal(t, "docker", cg.Runtime)
	assert.NotEmpty(t, cg.ContainerID)
}

func TestExtractHexSegment(t *testing.T) {
	// Returns the full leading hex run (>= 12 chars); no truncation.
	assert.Equal(t, "0123456789abcdef", extractHexSegment("0123456789abcdef"))
	assert.Equal(t, "0123456789abcdef", extractHexSegment("/0123456789abcdef.scope"))
	assert.Empty(t, extractHexSegment("short"))
	assert.Empty(t, extractHexSegment("/ZZZ"))
}

func TestIsHexByte(t *testing.T) {
	assert.True(t, isHexByte('0'))
	assert.True(t, isHexByte('a'))
	assert.True(t, isHexByte('F'))
	assert.False(t, isHexByte('g'))
	assert.False(t, isHexByte('/'))
}
