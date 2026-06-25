package app

import (
	"github.com/pmtop/pmtop/pkg/netstat"
)

// fakeSource is a test DataSource returning a fixed snapshot.
type fakeSource struct {
	socks []netstat.SocketInfo
	err   error
	calls int
}

func (f *fakeSource) Collect() ([]netstat.SocketInfo, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	// Return a copy so sorts don't mutate the caller's fixture.
	out := make([]netstat.SocketInfo, len(f.socks))
	copy(out, f.socks)
	return out, nil
}

// sampleSockets returns a small, deterministic set of sockets for tests.
func sampleSockets() []netstat.SocketInfo {
	return []netstat.SocketInfo{
		{Protocol: netstat.ProtocolTCP, LocalAddr: "0.0.0.0", LocalPort: 22, State: netstat.StateListen, Inode: 100, PID: 100, ProcessName: "sshd", User: "root"},
		{Protocol: netstat.ProtocolTCP, LocalAddr: "127.0.0.1", LocalPort: 8080, State: netstat.StateListen, Inode: 200, PID: 200, ProcessName: "nginx", User: "www-data", Runtime: "docker", ContainerID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{Protocol: netstat.ProtocolTCP, LocalAddr: "127.0.0.1", LocalPort: 8080, RemoteAddr: "127.0.0.1", RemotePort: 1234, State: netstat.StateEstablished, Inode: 300, PID: 300, ProcessName: "myapp", User: "user"},
		{Protocol: netstat.ProtocolUDP, LocalAddr: "0.0.0.0", LocalPort: 53, State: netstat.StateClose, Inode: 400, PID: 400, ProcessName: "dnsmasq", User: "dnsmasq"},
	}
}
