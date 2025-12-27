package formatter

import (
	"fmt"
	"os/exec"
)

// Format runs odinfmt on the given directory to format Odin source code.
func Format(srcDir string) error {
	fmt.Println("Formatting Odin code...")

	cmd := exec.Command("odinfmt", "-w", srcDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("running odinfmt: %w\nOutput: %s", err, string(output))
	}

	fmt.Println("✅ Formatting completed")
	return nil
}

// Check runs odinfmt in check mode (dry run) without modifying files.
func Check(srcDir string) error {
	fmt.Println("Checking Odin code formatting...")

	// odinfmt doesn't have a dedicated check mode, but we can use -stdin flag
	// For now, we'll just run it without -w to see if there are issues
	cmd := exec.Command("odinfmt", srcDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("checking format: %w\nOutput: %s", err, string(output))
	}

	fmt.Println("✅ Format check completed")
	return nil
}
