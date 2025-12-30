package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "venture",
	Short: "Venture - Build tool for the Adventurer game engine",
	Long: `Venture is a comprehensive build tool for the Adventurer game engine.
It handles import linting, protobuf generation, C library compilation,
Odin compilation, and distribution packaging for multiple platforms.`,
	SilenceUsage:      true,
	DisableAutoGenTag: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
