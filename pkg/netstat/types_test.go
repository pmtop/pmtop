package netstat

import "testing"

func TestProtocolHelpers(t *testing.T) {
	cases := []struct {
		p                Protocol
		tcp, udp, ipv6   bool
	}{
		{ProtocolTCP, true, false, false},
		{ProtocolTCP6, true, false, true},
		{ProtocolUDP, false, true, false},
		{ProtocolUDP6, false, true, true},
		{ProtocolRaw, false, false, false},
		{ProtocolRaw6, false, false, true},
		{ProtocolUnix, false, false, false},
	}
	for _, c := range cases {
		if c.p.IsTCP() != c.tcp {
			t.Errorf("%s IsTCP=%v want %v", c.p, c.p.IsTCP(), c.tcp)
		}
		if c.p.IsUDP() != c.udp {
			t.Errorf("%s IsUDP=%v want %v", c.p, c.p.IsUDP(), c.udp)
		}
		if c.p.IsIPv6() != c.ipv6 {
			t.Errorf("%s IsIPv6=%v want %v", c.p, c.p.IsIPv6(), c.ipv6)
		}
	}
}

func TestStateString(t *testing.T) {
	cases := []struct {
		s    State
		short, long, sym string
	}{
		{StateListen, "LISTEN", "LISTEN", "▶"},
		{StateEstablished, "ESTAB", "ESTABLISHED", "●"},
		{StateTimeWait, "TIME_WAIT", "TIME_WAIT", "▲"},
		{StateCloseWait, "CLOSE_WAIT", "CLOSE_WAIT", "▼"},
		{StateSynSent, "SYN_SENT", "SYN_SENT", "◆"},
		{StateClosing, "CLOSING", "CLOSING", "◀"},
		{StateUnconnected, "UNCONN", "UNCONNECTED", "-"},
		{StateUnknown, "UNKNOWN", "UNKNOWN", "-"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.short {
			t.Errorf("String(%v)=%q want %q", c.s, got, c.short)
		}
		if got := c.s.LongString(); got != c.long {
			t.Errorf("LongString(%v)=%q want %q", c.s, got, c.long)
		}
		if got := c.s.Symbol(); got != c.sym {
			t.Errorf("Symbol(%v)=%q want %q", c.s, got, c.sym)
		}
	}
}

func TestTCPStateFromHex(t *testing.T) {
	want := map[string]State{
		"01": StateEstablished, "02": StateSynSent, "03": StateSynRecv,
		"04": StateFinWait1, "05": StateFinWait2, "06": StateTimeWait,
		"07": StateClose, "08": StateCloseWait, "09": StateLastAck,
		"0A": StateListen, "0a": StateListen, "0B": StateClosing, "0b": StateClosing,
		"FF": StateUnknown,
	}
	for hex, wantS := range want {
		if got := TCPStateFromHex(hex); got != wantS {
			t.Errorf("TCPStateFromHex(%q)=%v want %v", hex, got, wantS)
		}
	}
}

func TestUnixStateFromCode(t *testing.T) {
	if UnixStateFromCode("01") != StateUnconnected {
		t.Error("01 should be Unconnected")
	}
	if UnixStateFromCode("02") != StateConnecting {
		t.Error("02 should be Connecting")
	}
	if UnixStateFromCode("03") != StateConnected {
		t.Error("03 should be Connected")
	}
	if UnixStateFromCode("99") != StateUnknown {
		t.Error("99 should be Unknown")
	}
}
