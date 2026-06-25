package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pmtop/pmtop/internal/collector"
	"github.com/pmtop/pmtop/internal/export"
	"github.com/pmtop/pmtop/internal/filter"
	"github.com/pmtop/pmtop/internal/platform"
	"github.com/pmtop/pmtop/pkg/netstat"
)

// listFlags holds the filter/output flags for `pmtop list`.
type listFlags struct {
	jsonOut    bool
	csvOut     bool
	tsvOut     bool
	proto      string
	state      string
	ports      string
	process    string
	pid        int
	user       string
	container  string
	localCIDR  string
	remoteCIDR string
	text       string
}

var listF listFlags

var listCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List ports and connections",
	Long: `List all TCP, UDP, and Unix domain sockets along with their
associated processes and container information.

By default, pmtop list outputs a tab-separated table suitable for
piping through awk, cut, or grep. Use --json or --csv for structured
output that can be consumed by scripts and monitoring tools.

The filter flags accept the same syntax as the TUI filter form:
  --ports 80,443,8080-8090   filter by port number or range
  --proto tcp,udp            filter by protocol
  --state LISTEN,ESTAB       filter by connection state
  --process nginx            fuzzy match process name
  --pid 1234                 match a specific PID
  --user root                fuzzy match user
  --container web            fuzzy match container name/id
  --local-cidr 192.168.1.0/24  filter local address by CIDR
  --remote-cidr 10.0.0.0/8     filter remote address by CIDR
  --text ssh                 free-text match across process/user/pid/container`,
	Example: `  # List all listening TCP ports as JSON
  pmtop list --proto tcp --state LISTEN --json

  # Filter ports 8080-8090 used by processes named "java"
  pmtop list --ports 8080-8090 --process java

  # Export full port table as CSV
  pmtop list --csv > ports.csv

  # Pipe-friendly tab-separated output
  pmtop list | awk -F'\t' '{print $6, $8}'`,
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.BoolVar(&listF.jsonOut, "json", false, "output as JSON")
	f.BoolVar(&listF.csvOut, "csv", false, "output as CSV with headers")
	f.BoolVar(&listF.tsvOut, "tsv", false, "output as tab-separated values (default)")
	f.StringVar(&listF.proto, "proto", "", "filter by protocol (tcp,tcp6,udp,udp6,raw,unix)")
	f.StringVar(&listF.state, "state", "", "filter by state (LISTEN,ESTAB,TIME_WAIT,...)")
	f.StringVar(&listF.ports, "ports", "", "filter by ports (80,80,443,8080-8090)")
	f.StringVar(&listF.process, "process", "", "fuzzy match process name")
	f.IntVar(&listF.pid, "pid", 0, "filter by PID")
	f.StringVar(&listF.user, "user", "", "fuzzy match user")
	f.StringVar(&listF.container, "container", "", "fuzzy match container name/id")
	f.StringVar(&listF.localCIDR, "local-cidr", "", "filter local address by CIDR")
	f.StringVar(&listF.remoteCIDR, "remote-cidr", "", "filter remote address by CIDR")
	f.StringVar(&listF.text, "text", "", "free-text match across process/user/pid/container")
	rootCmd.AddCommand(listCmd)
}

// buildListFilter converts listFlags into a filter.Filter, returning an error
// on the first invalid field.
func buildListFilter(f listFlags) (filter.Filter, error) {
	out := filter.Filter{Process: f.process, PID: f.pid, User: f.user, Container: f.container, Text: f.text}
	if f.ports != "" {
		p, err := filter.ParsePorts(f.ports)
		if err != nil {
			return out, fmt.Errorf("--ports: %w", err)
		}
		out.Ports = p
	}
	if f.proto != "" {
		p, err := filter.ParseProtocols(f.proto)
		if err != nil {
			return out, fmt.Errorf("--proto: %w", err)
		}
		out.Protocols = p
	}
	if f.state != "" {
		s, err := filter.ParseStates(f.state)
		if err != nil {
			return out, fmt.Errorf("--state: %w", err)
		}
		out.States = s
	}
	if f.localCIDR != "" {
		c, err := filter.ParseCIDR(f.localCIDR)
		if err != nil {
			return out, fmt.Errorf("--local-cidr: %w", err)
		}
		out.LocalCIDR = c
	}
	if f.remoteCIDR != "" {
		c, err := filter.ParseCIDR(f.remoteCIDR)
		if err != nil {
			return out, fmt.Errorf("--remote-cidr: %w", err)
		}
		out.RemoteCIDR = c
	}
	return out, nil
}

// runList collects sockets, applies filters, and prints in the chosen format.
func runList(cmd *cobra.Command, _ []string) error {
	if !platform.IsLinux() {
		return fmt.Errorf("pmtop list requires Linux /proc")
	}
	f, err := buildListFilter(listF)
	if err != nil {
		return err
	}
	src := collector.New(collector.NewOSFS(), collector.DefaultProcRoot)
	socks, err := src.Collect()
	if err != nil {
		return err
	}
	socks = filter.Apply(socks, f)

	switch {
	case listF.jsonOut:
		b, err := export.JSON(socks)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
	case listF.csvOut:
		b, err := export.CSV(socks)
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), string(b))
	default:
		// TSV is the default (FR-09-01).
		fmt.Fprint(cmd.OutOrStdout(), string(export.TSV(socks)))
	}
	return nil
}

// formatSocketRow renders a single socket as a human-readable line (used by
// info/list helpers). Kept here to avoid a UI dependency in the CLI path.
func formatSocketRow(s netstat.SocketInfo) string {
	container := s.ContainerName
	if container == "" && s.ContainerID != "" {
		container = s.ContainerID[:min12(len(s.ContainerID))]
	}
	if container == "" {
		container = "-"
	}
	proc := s.ProcessName
	user := s.User
	if s.PID == 0 {
		proc, user, container = "-", "-", "-"
	}
	return strings.Join([]string{
		string(s.Protocol),
		s.LocalAddr + ":" + strconv.Itoa(int(s.LocalPort)),
		s.RemoteAddr + ":" + strconv.Itoa(int(s.RemotePort)),
		s.State.String(),
		strconv.Itoa(s.PID),
		proc, user, container,
	}, "\t")
}

func min12(n int) int {
	if n > 12 {
		return 12
	}
	return n
}
