package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bloodmagesoftware/venture/clay"
	"github.com/bloodmagesoftware/venture/linter"
	"github.com/bloodmagesoftware/venture/odin"
	"github.com/bloodmagesoftware/venture/platform"
	"github.com/bloodmagesoftware/venture/project"
	"github.com/bloodmagesoftware/venture/protobuf"
	"github.com/bloodmagesoftware/venture/steamworks"
	"github.com/spf13/cobra"
)

var (
	runPlatform string
	runDebug    bool
	runRelease  bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Build and run the project for the current platform",
	Long:  `Builds the project for the current platform and runs it immediately. This is intended for development.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := getProjectRoot()
		if err != nil {
			return fmt.Errorf("getting project root: %w", err)
		}

		// Load project configuration
		config, err := project.LoadConfig(projectRoot)
		if err != nil {
			return fmt.Errorf("loading project config: %w", err)
		}

		// Detect current platform
		currentTarget, err := platform.DetectCurrent()
		if err != nil {
			return fmt.Errorf("detecting current platform: %w", err)
		}

		// Step 1: Lint
		srcDir := filepath.Join(projectRoot, "src")
		vendorDir := filepath.Join(projectRoot, "vendor")
		if err := linter.Lint(srcDir, vendorDir); err != nil {
			return fmt.Errorf("linting: %w", err)
		}

		// Step 2: Generate protobuf
		protoDir := filepath.Join(projectRoot, "proto")
		generatedDir := filepath.Join(srcDir, "generated")
		if err := protobuf.Generate(protoDir, generatedDir); err != nil {
			return fmt.Errorf("generating protobuf: %w", err)
		}

		// Step 3: Compile Clay
		clayDir := filepath.Join(projectRoot, "vendor", "clay")
		_, err = clay.Compile(clayDir, currentTarget)
		if err != nil {
			return fmt.Errorf("compiling clay: %w", err)
		}

		// Step 4: Ensure Steam libraries (if platform is steam)
		var steamLib *steamworks.LibraryInfo
		if runPlatform == "steam" {
			steamworksDir := filepath.Join(projectRoot, "vendor", "steamworks", "redistributable_bin")
			steamLib, err = steamworks.EnsureLibraries(currentTarget, steamworksDir)
			if err != nil {
				fmt.Printf("Warning: Steam libraries not available: %v\n", err)
			}
		}

		// Step 5: Compile Odin
		outputName := platform.GetOutputName(currentTarget, config.BinaryName)
		outputPath := filepath.Join(projectRoot, outputName)
		platformCollectionPath := fmt.Sprintf("src/platforms/%s", runPlatform)

		compileConfig := odin.CompileConfig{
			SrcDir:         srcDir,
			OutputPath:     outputPath,
			Target:         currentTarget,
			Platform:       runPlatform,
			Debug:          runDebug,
			Release:        runRelease,
			CollectionPath: platformCollectionPath,
		}

		if err := odin.Compile(compileConfig); err != nil {
			return fmt.Errorf("compiling odin: %w", err)
		}

		// Step 6: Copy Steam library to project root (if steam platform)
		var copiedLibPath string
		if runPlatform == "steam" && steamLib != nil {
			libName := filepath.Base(steamLib.RuntimeLib)
			copiedLibPath = filepath.Join(projectRoot, libName)

			fmt.Printf("Copying %s...\n", libName)
			if err := copyFile(steamLib.RuntimeLib, copiedLibPath); err != nil {
				fmt.Printf("Warning: Failed to copy Steam library: %v\n", err)
			}
		}

		// Step 7: Execute binary
		fmt.Printf("\nRunning %s...\n", outputPath)
		fmt.Println("----------------------------------------")

		runCmd := exec.Command(outputPath)
		runCmd.Dir = projectRoot
		runCmd.Stdout = os.Stdout
		runCmd.Stderr = os.Stderr
		runCmd.Stdin = os.Stdin

		// Set SteamAppId environment variable if running with Steam platform
		if runPlatform == "steam" {
			if config.SteamAppID == "" {
				fmt.Println("Warning: steam_app_id not set in venture.yaml")
			} else {
				runCmd.Env = append(os.Environ(), fmt.Sprintf("SteamAppId=%s", config.SteamAppID))
				fmt.Printf("Setting SteamAppId=%s\n", config.SteamAppID)
			}
		}

		runErr := runCmd.Run()

		// Step 8: Clean up binary
		fmt.Printf("\nCleaning up %s...\n", filepath.Base(outputPath))
		if err := os.Remove(outputPath); err != nil {
			fmt.Printf("Warning: Failed to clean up binary: %v\n", err)
		}

		// Step 9: Clean up Steam library after exit
		if copiedLibPath != "" {
			fmt.Printf("Cleaning up %s...\n", filepath.Base(copiedLibPath))
			if err := os.Remove(copiedLibPath); err != nil {
				fmt.Printf("Warning: Failed to clean up Steam library: %v\n", err)
			}
		}

		if runErr != nil {
			return fmt.Errorf("running binary: %w", runErr)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&runPlatform, "platform", "p", "fallback", "Platform (steam/fallback)")
	runCmd.Flags().BoolVarP(&runDebug, "debug", "d", false, "Build with debug symbols")
	runCmd.Flags().BoolVarP(&runRelease, "release", "r", false, "Build with optimizations")
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading source file: %w", err)
	}

	if err := os.WriteFile(dst, input, 0644); err != nil {
		return fmt.Errorf("writing destination file: %w", err)
	}

	// Preserve executable permissions if source has them
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("getting source file info: %w", err)
	}

	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	return nil
}
