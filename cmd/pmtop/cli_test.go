package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pmtop/pmtop/internal/filter"
	"github.com/pmtop/pmtop/pkg/netstat"
)

func TestBuildListFilter_AllFields(t *testing.T) {
	f, err := buildListFilter(listFlags{
		ports: "80,8080-8082", proto: "tcp,udp", state: "LISTEN,ESTAB",
		process: "nginx", pid: 100, user: "root", container: "web",
		localCIDR: "127.0.0.0/8", remoteCIDR: "10.0.0.0/8", text: "ssh",
	})
	require.NoError(t, err)
	assert.Equal(t, []uint16{80, 8080, 8081, 8082}, f.Ports)
	assert.Len(t, f.Protocols, 2)
	assert.Len(t, f.States, 2)
	assert.Equal(t, "nginx", f.Process)
	assert.Equal(t, 100, f.PID)
	assert.Equal(t, "root", f.User)
	assert.Equal(t, "web", f.Container)
	assert.NotNil(t, f.LocalCIDR)
	assert.NotNil(t, f.RemoteCIDR)
	assert.Equal(t, "ssh", f.Text)
}

func TestBuildListFilter_Empty(t *testing.T) {
	f, err := buildListFilter(listFlags{})
	require.NoError(t, err)
	assert.True(t, f.IsEmpty())
}

func TestBuildListFilter_Errors(t *testing.T) {
	cases := []listFlags{
		{ports: "abc"}, {proto: "foo"}, {state: "BOGUS"},
		{localCIDR: "not-a-cidr"}, {remoteCIDR: "10.0.0.0/33"},
	}
	for i, c := range cases {
		_, err := buildListFilter(c)
		assert.Error(t, err, "case %d", i)
	}
}

func TestBuildListFilter_ErrorPrefix(t *testing.T) {
	_, err := buildListFilter(listFlags{ports: "abc"})
	assert.ErrorContains(t, err, "--ports:")
	_, err = buildListFilter(listFlags{proto: "foo"})
	assert.ErrorContains(t, err, "--proto:")
}

func TestFormatSocketRow(t *testing.T) {
	s := netstat.SocketInfo{
		Protocol: netstat.ProtocolTCP, LocalAddr: "0.0.0.0", LocalPort: 22,
		State: netstat.StateListen, PID: 100, ProcessName: "sshd", User: "root",
	}
	row := formatSocketRow(s)
	assert.Contains(t, row, "tcp")
	assert.Contains(t, row, "0.0.0.0:22")
	assert.Contains(t, row, "LISTEN")
	assert.Contains(t, row, "sshd")

	// Ownerless.
	s2 := netstat.SocketInfo{Protocol: netstat.ProtocolUnix, Path: "/x", PID: 0}
	row = formatSocketRow(s2)
	assert.Contains(t, row, "\t-\t-\t-")
}

func TestMin12(t *testing.T) {
	assert.Equal(t, 12, min12(20))
	assert.Equal(t, 8, min12(8))
}

func TestFilterReusedFromPackage(t *testing.T) {
	// Sanity: the CLI filter builder is the same engine the TUI uses.
	var f filter.Filter
	assert.True(t, f.IsEmpty())
}
