package app

import (
	"time"

	"github.com/pmtop/pmtop/internal/collector"
	"github.com/pmtop/pmtop/internal/process"
	"github.com/pmtop/pmtop/pkg/netstat"
)

// DetailProvider supplies on-demand process and cgroup detail for the side
// panel. *collector.Collector satisfies this interface.
type DetailProvider interface {
	ProcessDetail(pid int) (collector.ProcessInfo, error)
	CgroupDetail(pid int) (collector.CgroupInfo, error)
}

// SignalSender sends a signal to a process. Abstracted so tests don't kill
// real processes.
type SignalSender interface {
	Send(pid int, sig process.Signal) error
}

// realSender delegates to the platform process.SendSignal.
type realSender struct{}

func (realSender) Send(pid int, sig process.Signal) error {
	return process.SendSignal(pid, sig)
}

// DetailState holds the rendered process-detail side panel content.
type DetailState struct {
	pid      int
	proc     collector.ProcessInfo
	cg       collector.CgroupInfo
	pkgName  string
	pkgErr   error
	err      error
	ready    bool
	scroll   int // scroll offset for long content

	// CPU% tracking (sampled across refreshes).
	prevUTime uint64
	prevSTime uint64
	prevTime  time.Time
	cpuPct    float64
}

// SignalState holds the signal-selection dialog state.
type SignalState struct {
	pid     int
	name    string
	sel     int
	confirm bool
	result  string
}

// openDetail fetches process/cgroup/package info for the selected socket and
// opens the detail panel. Synchronous /proc reads are fast enough for M4.
func (m *Model) openDetail() {
	s, ok := m.currentSocket()
	if !ok || s.PID <= 0 {
		m.setStatus("no process for this socket", 2*time.Second)
		return
	}
	d := &DetailState{pid: s.PID}
	dp, ok := m.source.(DetailProvider)
	if !ok {
		d.err = errDetailUnavailable
		m.detail = d
		m.mode = modeDetail
		return
	}
	if pi, err := dp.ProcessDetail(s.PID); err == nil {
		d.proc = pi
		d.prevUTime = pi.UTime
		d.prevSTime = pi.STime
		d.prevTime = time.Now()
	} else {
		d.err = err
	}
	if cg, err := dp.CgroupDetail(s.PID); err == nil {
		d.cg = cg
	}
	// Owning package of the executable (best-effort).
	if d.proc.Exe != "" {
		if name, _, err := process.PackageOwner(d.proc.Exe); err == nil {
			d.pkgName = name
		} else {
			d.pkgErr = err
		}
	}
	d.ready = true
	m.detail = d
	m.mode = modeDetail
}

// updateDetailData refreshes the process info for the open detail panel and
// computes CPU% from the delta since the last sample.
func (m *Model) updateDetailData() {
	if m.detail == nil {
		return
	}
	dp, ok := m.source.(DetailProvider)
	if !ok {
		return
	}
	pi, err := dp.ProcessDetail(m.detail.pid)
	if err != nil {
		return
	}
	// Compute CPU% from tick delta.
	total := pi.UTime + pi.STime
	prevTotal := m.detail.prevUTime + m.detail.prevSTime
	elapsed := time.Since(m.detail.prevTime).Seconds()
	if elapsed > 0 && prevTotal > 0 {
		tickDelta := float64(total - prevTotal)
		m.detail.cpuPct = tickDelta / float64(collector.DefaultHZ) / elapsed * 100.0
	}
	m.detail.prevUTime = pi.UTime
	m.detail.prevSTime = pi.STime
	m.detail.prevTime = time.Now()
	m.detail.proc = pi
}

// openSignal opens the signal-selection dialog for the selected process.
func (m *Model) openSignal() {
	s, ok := m.currentSocket()
	if !ok || s.PID <= 0 {
		m.setStatus("no process for this socket", 2*time.Second)
		return
	}
	m.signal = &SignalState{pid: s.PID, name: s.ProcessName, sel: defaultSignalIndex()}
	m.mode = modeSignal
}

// defaultSignalIndex returns the index of SIGTERM in process.Signals.
func defaultSignalIndex() int {
	for i, s := range process.Signals {
		if s.Name == "SIGTERM" {
			return i
		}
	}
	return 0
}

// sendCurrentSignal dispatches the selected signal and records the result.
func (m *Model) sendCurrentSignal() {
	if m.signal == nil || m.sender == nil {
		return
	}
	sig := process.Signals[m.signal.sel]
	if err := m.sender.Send(m.signal.pid, sig); err != nil {
		m.signal.result = "failed: " + err.Error()
	} else {
		m.signal.result = "sent " + sig.Name + " to " + m.signal.name + " (pid " + itoa(m.signal.pid) + ")"
	}
}

// itoa is a small allocation-free int -> string converter.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// errDetailUnavailable is returned when the data source cannot provide detail.
var errDetailUnavailable = detailErr("detail unavailable")

type detailErr string

func (e detailErr) Error() string { return string(e) }

// currentContainerInfo returns the container fields for the detail panel's PID,
// looking up the socket by PID rather than cursor position (fixes Q4).
func (m Model) currentContainerInfo() netstat.SocketInfo {
	if m.detail == nil {
		return netstat.SocketInfo{}
	}
	for _, s := range m.socks {
		if s.PID == m.detail.pid {
			return s
		}
	}
	return netstat.SocketInfo{}
}
