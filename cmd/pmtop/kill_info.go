package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/pmtop/pmtop/internal/collector"
	"github.com/pmtop/pmtop/internal/platform"
	"github.com/pmtop/pmtop/internal/process"
)

var (
	killSignal string
)

var killCmd = &cobra.Command{
	Use:   "kill <PID>",
	Short: "Send a signal to a process",
	Long: `Send a signal to a process identified by PID.

By default pmtop kill sends SIGTERM (signal 15). Use --signal to choose a
different signal by name (SIGKILL, SIGHUP, ...) or number (9, 1, ...).

This is the non-interactive counterpart to the TUI signal dialog. There is
no confirmation prompt; use with care.`,
	Example: `  # Gracefully stop a process (default SIGTERM)
  pmtop kill 1234

  # Force kill
  pmtop kill 1234 --signal SIGKILL
  pmtop kill 1234 --signal 9

  # Reload a daemon's config
  pmtop kill $(pidof nginx) --signal SIGHUP`,
	Args: cobra.ExactArgs(1),
	RunE: runKill,
}

func init() {
	killCmd.Flags().StringVarP(&killSignal, "signal", "s", "SIGTERM", "signal name or number (SIGTERM, SIGKILL, SIGHUP, ...)")
	rootCmd.AddCommand(killCmd)
}

// runKill parses the PID and signal, validates them, and sends the signal.
func runKill(cmd *cobra.Command, args []string) error {
	if !platform.IsLinux() {
		return fmt.Errorf("pmtop kill requires Linux")
	}
	pid, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PID %q: %w", args[0], err)
	}
	if err := process.ValidatePID(pid); err != nil {
		return err
	}
	sig, ok := process.ParseSignal(killSignal)
	if !ok {
		return fmt.Errorf("unknown signal %q", killSignal)
	}
	if err := process.SendSignal(pid, sig); err != nil {
		return fmt.Errorf("failed to send %s to %d: %w", sig.Name, pid, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "sent %s to PID %d\n", sig.Name, pid)
	return nil
}

var infoCmd = &cobra.Command{
	Use:   "info <PID>",
	Short: "Print process detail",
	Long: `Print detailed information about a process: PID, PPID, name, full
command line, executable path, working directory, start time, CPU/memory
usage, user/group, cgroup, and container association.

Output is structured text by default; use --json for machine-readable JSON.`,
	Example: `  # Inspect a process
  pmtop info 1234

  # Machine-readable
  pmtop info 1234 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runInfo,
}

func init() {
	infoCmd.Flags().BoolVar(&infoF.jsonOut, "json", false, "output as JSON")
	rootCmd.AddCommand(infoCmd)
}

// infoFlags holds flags for `pmtop info`.
var infoF struct{ jsonOut bool }

// runInfo collects and prints process detail for a PID.
func runInfo(cmd *cobra.Command, args []string) error {
	if !platform.IsLinux() {
		return fmt.Errorf("pmtop info requires Linux /proc")
	}
	pid, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PID %q: %w", args[0], err)
	}
	if err := process.ValidatePID(pid); err != nil {
		return err
	}
	src := collector.New(collector.NewOSFS(), collector.DefaultProcRoot)
	pi, err := src.ProcessDetail(pid)
	if err != nil {
		return fmt.Errorf("process %d: %w", pid, err)
	}
	cg, _ := src.CgroupDetail(pid)
	if infoF.jsonOut {
		return printProcessJSON(cmd, pi, cg)
	}
	printProcessText(cmd, pi, cg)
	return nil
}
