package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/pmtop/pmtop/internal/collector"
)

// printProcessText renders a ProcessInfo + CgroupInfo as human-readable text.
func printProcessText(cmd *cobra.Command, pi collector.ProcessInfo, cg collector.CgroupInfo) {
	w := func(k, v string) { fmt.Fprintf(cmd.OutOrStdout(), "%-12s %s\n", k, v) }
	out := cmd.OutOrStdout()
	w("PID:", strconv.Itoa(pi.PID))
	w("PPID:", strconv.Itoa(pi.PPID))
	w("Name:", pi.Name)
	w("State:", pi.State)
	w("User:", pi.User+" ("+strconv.Itoa(int(pi.UID))+")")
	w("Group:", pi.Group+" ("+strconv.Itoa(int(pi.GID))+")")
	w("Command:", pi.Cmdline)
	w("Exe:", pi.Exe)
	w("CWD:", pi.CWD)
	w("Start:", pi.StartTime.String())
	w("VmRSS:", strconv.FormatUint(pi.VmRSS, 10)+" B")
	w("VmSize:", strconv.FormatUint(pi.VmSize, 10)+" B")
	if cg.Runtime != "" {
		w("Runtime:", cg.Runtime)
		w("Container:", cg.ContainerID)
	}
	if len(cg.Lines) > 0 {
		fmt.Fprintf(out, "%-12s v%d %s\n", "Cgroup:", cg.Version, cg.Lines[0].Path)
	}
}

// processJSON is the JSON projection of ProcessInfo + CgroupInfo for `pmtop info --json`.
type processJSON struct {
	PID        int    `json:"pid"`
	PPID       int    `json:"ppid"`
	Name       string `json:"name"`
	State      string `json:"state"`
	UID        uint32 `json:"uid"`
	User       string `json:"user"`
	GID        uint32 `json:"gid"`
	Group      string `json:"group"`
	Cmdline    string `json:"cmdline"`
	Exe        string `json:"exe"`
	CWD        string `json:"cwd"`
	StartTime  string `json:"start_time"`
	VmRSS      uint64 `json:"vm_rss"`
	VmSize     uint64 `json:"vm_size"`
	CgVersion  int    `json:"cgroup_version,omitempty"`
	Runtime    string `json:"runtime,omitempty"`
	Container  string `json:"container_id,omitempty"`
	CgroupPath string `json:"cgroup_path,omitempty"`
}

// printProcessJSON renders the process detail as indented JSON.
func printProcessJSON(cmd *cobra.Command, pi collector.ProcessInfo, cg collector.CgroupInfo) error {
	j := processJSON{
		PID: pi.PID, PPID: pi.PPID, Name: pi.Name, State: pi.State,
		UID: pi.UID, User: pi.User, GID: pi.GID, Group: pi.Group,
		Cmdline: pi.Cmdline, Exe: pi.Exe, CWD: pi.CWD,
		StartTime: pi.StartTime.String(),
		VmRSS:     pi.VmRSS, VmSize: pi.VmSize,
		CgVersion: cg.Version, Runtime: cg.Runtime, Container: cg.ContainerID,
	}
	if len(cg.Lines) > 0 {
		j.CgroupPath = cg.Lines[0].Path
	}
	b, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return nil
}
