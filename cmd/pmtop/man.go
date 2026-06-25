package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/pmtop/pmtop/internal/version"
)

// manCmd generates man pages from the cobra command tree (PRD 8.2). Used by
// `make man` and the goreleaser before hook.
var manCmd = &cobra.Command{
	Use:   "man --output-dir <dir>",
	Short: "Generate man pages",
	Long: `Generate roff man pages for pmtop and its subcommands from the cobra
command tree, using cobra's GenManTree. The output directory must exist or be
creatable.

Two pages are produced: pmtop(8) for the tool and pmtop.toml(5) for the
configuration file (when a config man section is registered).`,
	Example: `  # Generate man pages into ./man
  pmtop man --output-dir man

  # During release builds (called by Makefile)
  make man`,
	RunE: runMan,
}

var manOutputDir string

func init() {
	manCmd.Flags().StringVar(&manOutputDir, "output-dir", "man", "output directory for man pages")
	rootCmd.AddCommand(manCmd)
}

// runMan generates the man tree. The header (section/date/source) is set from
// the version package so generated pages stay reproducible.
func runMan(cmd *cobra.Command, _ []string) error {
	if manOutputDir == "" {
		return fmt.Errorf("--output-dir is required")
	}
	if err := os.MkdirAll(manOutputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	header := &doc.GenManHeader{
		Title:   "PMTOP",
		Section: "8",
		Source:  "pmtop " + version.Short(),
		Manual:  "System Administration",
	}
	if err := doc.GenManTree(rootCmd, header, manOutputDir); err != nil {
		return fmt.Errorf("gen man tree: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "man pages generated in %s\n", manOutputDir)
	return nil
}
