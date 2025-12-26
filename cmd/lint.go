package cmd

import (
	"path/filepath"

	"github.com/bloodmagesoftware/venture/linter"
	"github.com/bloodmagesoftware/venture/project"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint Odin source code for console portability",
	Long:  `Scans Odin source files for forbidden imports that prevent console portability.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get project root (parent of venture directory)
		projectRoot, err := getProjectRoot()
		if err != nil {
			return err
		}

		srcDir := filepath.Join(projectRoot, "src")
		vendorDir := filepath.Join(projectRoot, "vendor")

		if err := linter.Lint(srcDir, vendorDir); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(lintCmd)
}

// getProjectRoot returns the project root directory by looking for venture.yaml.
func getProjectRoot() (string, error) {
	return project.FindProjectRoot()
}

