// Package main is the pmtop binary entry point.
//
// pmtop is an interactive terminal UI for inspecting Linux ports, processes,
// and container associations. It reads kernel interfaces directly (/proc) and
// has zero external runtime dependencies.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/pmtop/pmtop/internal/app"
	"github.com/pmtop/pmtop/internal/collector"
	"github.com/pmtop/pmtop/internal/config"
	"github.com/pmtop/pmtop/internal/elevate"
	"github.com/pmtop/pmtop/internal/platform"
	"github.com/pmtop/pmtop/internal/version"
)

// Global flags parsed by cobra and applied to the config (FR-08-03).
var (
	flagNoElevate    bool
	flagInterval     string
	flagSortColumn   string
	flagSortAsc      bool
	flagColorblind   bool
	flagNoColor      bool
	flagDockerSocket string
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
	Example: `  # Start the interactive TUI (run as root for full visibility)
  sudo pmtop

  # Run as the current user only, no sudo banner (CI/automation)
  pmtop --no-elevate

  # Use a 1-second refresh interval
  pmtop --interval 1s

  # Non-interactive listing
  pmtop list --proto tcp --state LISTEN --json`,
	Version: version.Short(),
	RunE:    runTUI,
}

func init() {
	flags := rootCmd.PersistentFlags()
	flags.BoolVar(&flagNoElevate, "no-elevate", false, "force current-user mode, no sudo banner (CI/automation)")
	flags.StringVar(&flagInterval, "interval", "", "refresh interval (500ms,1s,2s,5s,manual)")
	flags.StringVar(&flagSortColumn, "sort", "", "default sort column (proto,port,state,pid,process,local,remote,container)")
	flags.BoolVar(&flagSortAsc, "asc", false, "sort ascending")
	flags.BoolVar(&flagColorblind, "colorblind", false, "enable colorblind-accessible symbol indicators")
	flags.BoolVar(&flagNoColor, "no-color", false, "disable all colors (also honored via NO_COLOR env)")
	flags.StringVar(&flagDockerSocket, "docker-socket", "", "Docker daemon socket path")
	rootCmd.AddCommand(versionCmd)
}

// loadCfg merges layered config with CLI flags (FR-08-03). It reads flag
// values from the root command's flag set.
func loadCfg(fs pflagFlagSet) (config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		// A malformed config file is non-fatal; fall back to defaults.
		cfg = config.Default()
	}
	cfg = config.ApplyFlags(cfg, config.Flags{
		RefreshInterval:    flagInterval,
		SortColumn:         flagSortColumn,
		SortAsc:            flagSortAsc,
		SortAscProvided:    fs.Changed("asc"),
		ColorblindMode:     flagColorblind,
		ColorblindProvided: fs.Changed("colorblind"),
		NoColor:            flagNoColor,
		DockerSocket:       flagDockerSocket,
	})
	return cfg, nil
}

// pflagFlagSet is the subset of *pflag.FlagSet used by loadCfg.
type pflagFlagSet interface {
	Changed(string) bool
}

// runTUI launches the interactive TUI using the real /proc collector.
func runTUI(cmd *cobra.Command, _ []string) error {
	if !platform.IsLinux() {
		return fmt.Errorf("pmtop TUI requires Linux /proc")
	}
	cfg, _ := loadCfg(cmd.PersistentFlags())

	// Privilege model (FR-07): detect, show banner for non-root unless
	// --no-elevate, and offer an opt-in sudo re-launch.
	st := elevate.Detect(flagNoElevate)
	if st.Restricted && shouldShowBanner() {
		if err := handleNonRoot(st); err != nil {
			return err
		}
	}

	interval := cfg.Interval()
	src := collector.New(
		collector.NewOSFS(), collector.DefaultProcRoot,
		collector.WithContainerResolver(collector.NewDockerResolver(cfg.DockerSocket)),
	)
	root := st.IsRoot
	m := app.New(src, version.Short(), root, interval)
	m.SetConfig(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

// shouldShowBanner reports whether the non-root banner should be presented
// (skipped when stdin is not a TTY, e.g. piped/CI).
func shouldShowBanner() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// handleNonRoot prints the restricted-mode banner and waits for a key. 'S'
// triggers an opt-in sudo re-launch; any other key continues restricted.
func handleNonRoot(st elevate.State) error {
	fmt.Fprintln(os.Stderr, elevate.BannerText())
	fmt.Fprint(os.Stderr, "> ")
	var key string
	_, _ = fmt.Scanln(&key)
	if len(key) > 0 && (key[0] == 'S' || key[0] == 's') {
		return elevate.Relaunch()
	}
	return nil
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the pmtop version",
	Long: `Print the pmtop version, commit, and build date as injected at link time.
Useful for bug reports and verifying installed builds.`,
	Example: `  pmtop version
  pmtop version --help`,
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintln(cmd.OutOrStdout(), version.String())
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
