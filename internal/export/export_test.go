package export

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pmtop/pmtop/pkg/netstat"
)

func sample() []netstat.SocketInfo {
	return []netstat.SocketInfo{
		{Protocol: netstat.ProtocolTCP, LocalAddr: "0.0.0.0", LocalPort: 22, State: netstat.StateListen, Inode: 1, PID: 100, ProcessName: "sshd", User: "root"},
		{Protocol: netstat.ProtocolTCP, LocalAddr: "127.0.0.1", LocalPort: 8080, State: netstat.StateListen, Inode: 2, PID: 200, ProcessName: "nginx", User: "www-data", Runtime: "docker", ContainerID: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", ContainerName: "web"},
		{Protocol: netstat.ProtocolUnix, Path: "/tmp/sock", State: netstat.StateUnconnected, Inode: 3, PID: 0},
	}
}

func TestFromSocket(t *testing.T) {
	r := FromSocket(sample()[1])
	assert.Equal(t, "tcp", r.Protocol)
	assert.Equal(t, 8080, r.LocalPort)
	assert.Equal(t, "LISTEN", r.State)
	assert.Equal(t, "web", r.Container)
	assert.Equal(t, "docker", r.Runtime)

	// Ownerless unix socket.
	r = FromSocket(sample()[2])
	assert.Equal(t, "-", r.Process)
	assert.Equal(t, "-", r.Container)
	assert.Equal(t, "UNCONN", r.State) // unix unconnected is a real state
}

func TestFromSocket_ShortContainerID(t *testing.T) {
	r := FromSocket(netstat.SocketInfo{Protocol: netstat.ProtocolTCP, PID: 1, ContainerID: "abc123def456", ProcessName: "x", User: "u"})
	assert.Equal(t, "abc123def456", r.Container, "short id used as-is")
}

func TestJSON(t *testing.T) {
	out, err := JSON(sample())
	require.NoError(t, err)
	var rows []Row
	require.NoError(t, json.Unmarshal(out, &rows))
	assert.Len(t, rows, 3)
	assert.Equal(t, "sshd", rows[0].Process)
	assert.Equal(t, "web", rows[1].Container)
	// No trailing newline (we trim it).
	assert.False(t, strings.HasSuffix(string(out), "\n\n"))
}

func TestCSV(t *testing.T) {
	out, err := CSV(sample())
	require.NoError(t, err)
	r := csv.NewReader(strings.NewReader(string(out)))
	records, err := r.ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 4) // header + 3
	assert.Equal(t, "Protocol", records[0][0])
	assert.Equal(t, "sshd", records[1][7])
	assert.Equal(t, "web", records[2][9])
}

func TestTSV(t *testing.T) {
	out := TSV(sample())
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	require.Len(t, lines, 3)
	// tab-separated, no header (FR-09-01).
	assert.Equal(t, 10, len(strings.Split(lines[0], "\t")))
	assert.Contains(t, lines[0], "sshd")
}

func TestJSON_Empty(t *testing.T) {
	out, err := JSON(nil)
	require.NoError(t, err)
	assert.Equal(t, "[]", string(out))
}

func TestCSV_Empty(t *testing.T) {
	out, err := CSV(nil)
	require.NoError(t, err)
	// header only
	r := csv.NewReader(strings.NewReader(string(out)))
	records, err := r.ReadAll()
	require.NoError(t, err)
	assert.Len(t, records, 1)
}

func TestTSV_Empty(t *testing.T) {
	assert.Empty(t, TSV(nil))
}
