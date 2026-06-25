package app

import (
	"github.com/pmtop/pmtop/internal/collector"
	"github.com/pmtop/pmtop/internal/process"
	"github.com/pmtop/pmtop/pkg/netstat"
)

// fakeSource is a test DataSource returning a fixed snapshot.
type fakeSource struct {
	socks []netstat.SocketInfo
	err   error
	calls int
	// optional detail overrides
	proc   map[int]collector.ProcessInfo
	procErr error
	cg     map[int]collector.CgroupInfo
}

func (f *fakeSource) Collect() ([]netstat.SocketInfo, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	out := make([]netstat.SocketInfo, len(f.socks))
	copy(out, f.socks)
	return out, nil
}

// ProcessDetail implements DetailProvider for tests.
func (f *fakeSource) ProcessDetail(pid int) (collector.ProcessInfo, error) {
	if f.procErr != nil {
		return collector.ProcessInfo{}, f.procErr
	}
	if pi, ok := f.proc[pid]; ok {
		return pi, nil
	}
	return collector.ProcessInfo{PID: pid, Name: "proc-" + itoa(pid)}, nil
}

// CgroupDetail implements DetailProvider for tests.
func (f *fakeSource) CgroupDetail(pid int) (collector.CgroupInfo, error) {
	if cg, ok := f.cg[pid]; ok {
		return cg, nil
	}
	return collector.CgroupInfo{Version: 2}, nil
}

// fakeSender records signals without sending them.
type fakeSender struct {
	sent   []sentSignal
	fail   error
}

type sentSignal struct {
	pid int
	sig process.Signal
}

func (s *fakeSender) Send(pid int, sig process.Signal) error {
	if s.fail != nil {
		return s.fail
	}
	s.sent = append(s.sent, sentSignal{pid: pid, sig: sig})
	return nil
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
