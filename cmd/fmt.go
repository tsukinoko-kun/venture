package cmd

import (
	"path/filepath"

	"github.com/bloodmagesoftware/venture/formatter"
	"github.com/spf13/cobra"
)

var (
	fmtCheck bool
)

var fmtCmd = &cobra.Command{
	Use:   "fmt",
	Short: "Format Odin source code",
	Long:  `Runs odinfmt on the Odin source code to format it consistently.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := getProjectRoot()
		if err != nil {
			return err
		}

		srcDir := filepath.Join(projectRoot, "src")

		if fmtCheck {
			return formatter.Check(srcDir)
		}

		return formatter.Format(srcDir)
	},
}

func init() {
	rootCmd.AddCommand(fmtCmd)
	fmtCmd.Flags().BoolVar(&fmtCheck, "check", false, "Check formatting without modifying files")
}
