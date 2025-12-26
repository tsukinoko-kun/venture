package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/bloodmagesoftware/venture/clay"
	"github.com/bloodmagesoftware/venture/linter"
	"github.com/bloodmagesoftware/venture/odin"
	"github.com/bloodmagesoftware/venture/packager"
	"github.com/bloodmagesoftware/venture/platform"
	"github.com/bloodmagesoftware/venture/project"
	"github.com/bloodmagesoftware/venture/protobuf"
	"github.com/bloodmagesoftware/venture/steamworks"
	"github.com/spf13/cobra"
)

var (
	buildTarget   string
	buildPlatform string
	buildDebug    bool
	buildRelease  bool
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build and package the project for distribution",
	Long:  `Builds the project for the specified target and creates a distribution zip package.`,
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

		// Determine target
		target := buildTarget
		if target == "" {
			target, err = platform.DetectCurrent()
			if err != nil {
				return fmt.Errorf("detecting current platform: %w", err)
			}
		} else {
			// Map friendly name to Odin target if needed
			target = platform.MapTarget(target)
		}

		fmt.Printf("Building for target: %s, platform: %s\n", target, buildPlatform)

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

		// Step 3: Compile Clay (with Zig for cross-compilation if available)
		clayDir := filepath.Join(projectRoot, "vendor", "clay")
		currentTarget, _ := platform.DetectCurrent()
		useZig := target != currentTarget
		
		_, err = clay.Compile(clayDir, target, useZig)
		if err != nil {
			return fmt.Errorf("compiling clay: %w", err)
		}

		// Step 4: Ensure Steam libraries (if platform is steam)
		var steamLib *steamworks.LibraryInfo
		if buildPlatform == "steam" {
			steamworksDir := filepath.Join(projectRoot, "vendor", "steamworks", "redistributable_bin")
			steamLib, err = steamworks.EnsureLibraries(target, steamworksDir)
			if err != nil {
				return fmt.Errorf("ensuring steam libraries: %w", err)
			}
		}

		// Step 5: Compile Odin
		outputName := platform.GetOutputName(target, config.BinaryName)
		outputPath := filepath.Join(projectRoot, outputName)
		platformCollectionPath := fmt.Sprintf("src/platforms/%s", buildPlatform)

		compileConfig := odin.CompileConfig{
			SrcDir:         srcDir,
			OutputPath:     outputPath,
			Target:         target,
			Platform:       buildPlatform,
			Debug:          buildDebug,
			Release:        buildRelease,
			CollectionPath: platformCollectionPath,
		}

		if err := odin.Compile(compileConfig); err != nil {
			return fmt.Errorf("compiling odin: %w", err)
		}

		// Step 6: Package for distribution
		assetsDir := filepath.Join(projectRoot, "assets")
		buildDir := filepath.Join(projectRoot, "build")

		var libraries []string
		if buildPlatform == "steam" && steamLib != nil {
			libraries = append(libraries, steamLib.RuntimeLib)
		}

		packageConfig := packager.PackageConfig{
			BinaryPath: outputPath,
			BinaryName: config.BinaryName,
			AssetsDir:  assetsDir,
			Libraries:  libraries,
			Target:     target,
			OutputDir:  buildDir,
		}

		zipPath, err := packager.Package(packageConfig)
		if err != nil {
			return fmt.Errorf("packaging: %w", err)
		}

		fmt.Printf("\n✅ Build complete: %s\n", zipPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringVarP(&buildTarget, "target", "t", "", "Target platform (defaults to current platform)")
	buildCmd.Flags().StringVarP(&buildPlatform, "platform", "p", "fallback", "Platform (steam/fallback)")
	buildCmd.Flags().BoolVarP(&buildDebug, "debug", "d", false, "Build with debug symbols")
	buildCmd.Flags().BoolVarP(&buildRelease, "release", "r", false, "Build with optimizations")
}

