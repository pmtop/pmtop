// Package export renders socket snapshots as JSON or CSV (PRD FR-08-05,
// FR-03-09). Pure functions over netstat.SocketInfo for full testability.
package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/pmtop/pmtop/pkg/netstat"
)

// Row is the export representation of a socket: a stable, human-friendly
// projection of netstat.SocketInfo with consistent field names.
type Row struct {
	Protocol       string `json:"protocol" csv:"Protocol"`
	LocalAddress   string `json:"local_address" csv:"LocalAddress"`
	LocalPort      int    `json:"local_port" csv:"LocalPort"`
	RemoteAddress  string `json:"remote_address" csv:"RemoteAddress"`
	RemotePort     int    `json:"remote_port" csv:"RemotePort"`
	State          string `json:"state" csv:"State"`
	PID            int    `json:"pid" csv:"PID"`
	Process        string `json:"process" csv:"Process"`
	User           string `json:"user" csv:"User"`
	Container      string `json:"container" csv:"Container"`
	ContainerImage string `json:"container_image,omitempty" csv:"ContainerImage"`
	Runtime        string `json:"runtime,omitempty" csv:"Runtime"`
	Inode          uint64 `json:"inode" csv:"Inode"`
}

// FromSocket converts a SocketInfo to an export Row.
func FromSocket(s netstat.SocketInfo) Row {
	container := s.ContainerName
	if container == "" && s.ContainerID != "" {
		if len(s.ContainerID) > 12 {
			container = s.ContainerID[:12]
		} else {
			container = s.ContainerID
		}
	}
	proc := s.ProcessName
	user := s.User
	if s.PID == 0 {
		proc, user, container = "-", "-", "-"
	}
	return Row{
		Protocol:       string(s.Protocol),
		LocalAddress:   s.LocalAddr,
		LocalPort:      int(s.LocalPort),
		RemoteAddress:  s.RemoteAddr,
		RemotePort:     int(s.RemotePort),
		State:          stateLabel(s),
		PID:            s.PID,
		Process:        proc,
		User:           user,
		Container:      container,
		ContainerImage: s.ContainerImage,
		Runtime:        s.Runtime,
		Inode:          s.Inode,
	}
}

// stateLabel returns the state name. UDP/raw sockets with Unknown state show
// "-"; unix and TCP states are always named.
func stateLabel(s netstat.SocketInfo) string {
	if !s.Protocol.IsTCP() && s.Protocol != netstat.ProtocolUnix && s.State == netstat.StateUnknown {
		return "-"
	}
	return s.State.String()
}

// JSON renders socks as pretty-printed JSON.
func JSON(socks []netstat.SocketInfo) ([]byte, error) {
	rows := make([]Row, len(socks))
	for i, s := range socks {
		rows[i] = FromSocket(s)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rows); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// CSV renders socks as CSV with a header row.
func CSV(socks []netstat.SocketInfo) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := []string{"Protocol", "LocalAddress", "LocalPort", "RemoteAddress",
		"RemotePort", "State", "PID", "Process", "User", "Container",
		"ContainerImage", "Runtime", "Inode"}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, s := range socks {
		r := FromSocket(s)
		if err := w.Write([]string{
			r.Protocol, r.LocalAddress, strconv.Itoa(r.LocalPort),
			r.RemoteAddress, strconv.Itoa(r.RemotePort), r.State,
			strconv.Itoa(r.PID), r.Process, r.User, r.Container,
			r.ContainerImage, r.Runtime, strconv.FormatUint(r.Inode, 10),
		}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// TSV renders socks as tab-separated values (no header) for piping (FR-09-01).
func TSV(socks []netstat.SocketInfo) []byte {
	var sb strings.Builder
	for _, s := range socks {
		r := FromSocket(s)
		sb.WriteString(strings.Join([]string{
			r.Protocol, r.LocalAddress, strconv.Itoa(r.LocalPort),
			r.RemoteAddress, strconv.Itoa(r.RemotePort), r.State,
			strconv.Itoa(r.PID), r.Process, r.User, r.Container,
		}, "\t"))
		sb.WriteByte('\n')
	}
	return []byte(sb.String())
}
