package odin

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// CompileConfig holds the configuration for Odin compilation.
type CompileConfig struct {
	SrcDir         string // Source directory
	OutputPath     string // Output binary path
	Target         string // Odin target (e.g., "darwin_arm64")
	Platform       string // Platform (e.g., "steam", "fallback")
	Debug          bool   // Enable debug mode
	Release        bool   // Enable release optimizations
	CollectionPath string // Path to platform collection (relative to project root)
}

// Compile builds the Odin project with the given configuration.
func Compile(config CompileConfig) error {
	fmt.Printf("Building Odin project for target: %s, platform: %s\n", config.Target, config.Platform)

	// Build command
	args := []string{
		"build",
		config.SrcDir,
		fmt.Sprintf("-out:%s", config.OutputPath),
		fmt.Sprintf("-target:%s", config.Target),
		fmt.Sprintf("-collection:platform=%s", config.CollectionPath),
	}

	if config.Debug {
		args = append(args, "-debug")
	} else if config.Release {
		args = append(args, "-o:speed")
	}

	fmt.Printf("Running: odin %s\n", joinArgs(args))

	cmd := exec.Command("odin", args...)
	cmd.Dir = filepath.Dir(config.SrcDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("odin compilation failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("Build successful: %s\n", config.OutputPath)
	return nil
}

// joinArgs joins command arguments for display purposes.
func joinArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += arg
	}
	return result
}

