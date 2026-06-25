// Package main is the pmtop binary entry point.
//
// pmtop is an interactive terminal UI for inspecting Linux ports, processes,
// and container associations. It reads kernel interfaces directly (/proc) and
// has zero external runtime dependencies.
package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/pmtop/pmtop/internal/app"
	"github.com/pmtop/pmtop/internal/collector"
	"github.com/pmtop/pmtop/internal/platform"
	"github.com/pmtop/pmtop/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "pmtop",
	Short: "Interactive Linux port & process manager",
	Long: `pmtop is an interactive terminal UI for inspecting Linux ports,
processes, and container associations from a single pane of glass.

It reads /proc directly (no ss/netstat/lsof/docker CLI dependencies) and
presents a unified, filterable, sortable view of every socket on the system
along with its owning process and container.

Run without a subcommand to start the interactive TUI. Use the "list",
"kill", and "info" subcommands for non-interactive, scriptable output.

See "pmtop help" and the man page pmtop(8) for full documentation.`,
	Version: version.Short(),
	RunE:    runTUI,
}

// runTUI launches the interactive TUI using the real /proc collector.
func runTUI(cmd *cobra.Command, _ []string) error {
	if !platform.IsLinux() {
		return fmt.Errorf("pmtop TUI requires Linux /proc")
	}
	src := collector.New(collector.NewOSFS(), collector.DefaultProcRoot)
	root := platform.CurrentUID() == 0
	m := app.New(src, version.Short(), root, 2*time.Second)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the pmtop version",
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintln(cmd.OutOrStdout(), version.String())
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
