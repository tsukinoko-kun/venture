package cmd

import (
	"context"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bloodmagesoftware/venture/bsp"
	"github.com/bloodmagesoftware/venture/clay"
	"github.com/bloodmagesoftware/venture/level"
	"github.com/bloodmagesoftware/venture/linter"
	"github.com/bloodmagesoftware/venture/odin"
	"github.com/bloodmagesoftware/venture/packager"
	"github.com/bloodmagesoftware/venture/platform"
	"github.com/bloodmagesoftware/venture/project"
	pb "github.com/bloodmagesoftware/venture/proto/level"
	"github.com/bloodmagesoftware/venture/protobuf"
	"github.com/bloodmagesoftware/venture/steamworks"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

var (
	buildPlatform string
	buildDebug    bool
	buildRelease  bool
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build and package the project for distribution",
	Long:  `Builds the project for the current OS and creates a distribution package (.app on macOS, bundled directory on Linux).`,
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

		// Detect current platform (no cross-compilation support)
		target, err := platform.DetectCurrent()
		if err != nil {
			return fmt.Errorf("detecting current platform: %w", err)
		}

		fmt.Printf("Building for current platform: %s, platform: %s\n", target, buildPlatform)

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

		// Step 2.5: Create level building iterator
		fmt.Println("Preparing level conversion with 30s timeout per level...")
		assetsDir := filepath.Join(projectRoot, "assets")
		levelIterator := buildLevelsIterator(assetsDir)

		// Step 3: Compile Clay
		clayDir := filepath.Join(projectRoot, "vendor", "clay")

		clayObject, err := clay.Compile(clayDir, target)
		if err != nil {
			return fmt.Errorf("compiling clay: %w", err)
		}
		fmt.Printf("Clay object file: %s\n", clayObject)

		// Step 4: Ensure Steam libraries (if platform is steam)
		fmt.Println("Checking Steam libraries...")
		var steamLib *steamworks.LibraryInfo
		if buildPlatform == "steam" {
			steamworksDir := filepath.Join(projectRoot, "vendor", "steamworks", "redistributable_bin")
			steamLib, err = steamworks.EnsureLibraries(target, steamworksDir)
			if err != nil {
				return fmt.Errorf("ensuring steam libraries: %w", err)
			}
		}

		// Step 5: Compile Odin
		fmt.Println("Starting Odin compilation...")
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
		fmt.Println("Odin compilation completed successfully")

		// Step 6: Package for distribution
		fmt.Println("Starting packaging...")
		buildDir := filepath.Join(projectRoot, "build")

		var libraries []string
		if buildPlatform == "steam" && steamLib != nil {
			libraries = append(libraries, steamLib.RuntimeLib)
		}

		packageConfig := packager.PackageConfig{
			ProjectRoot: projectRoot,
			BinaryPath:  outputPath,
			BinaryName:  config.BinaryName,
			AssetsDir:   assetsDir,
			Libraries:   libraries,
			LibraryVersions: packager.LibraryVersions{
				SDL:      config.Libraries.SDL,
				SDLTTF:   config.Libraries.SDLTTF,
				SDLImage: config.Libraries.SDLImage,
			},
			Target:        target,
			OutputDir:     buildDir,
			LevelIterator: levelIterator,
		}

		packagePath, err := packager.Package(packageConfig)
		if err != nil {
			return fmt.Errorf("packaging: %w", err)
		}

		fmt.Printf("\n✅ Build complete: %s\n", packagePath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringVarP(&buildPlatform, "platform", "p", "fallback", "Platform (steam/fallback)")
	buildCmd.Flags().BoolVarP(&buildDebug, "debug", "d", false, "Build with debug symbols")
	buildCmd.Flags().BoolVarP(&buildRelease, "release", "r", false, "Build with optimizations")
}

// convertLevelToProto converts a YAML level to protobuf format
func convertLevelToProto(yamlLevel *level.Level) (*pb.LevelData, error) {
	if yamlLevel == nil {
		return nil, fmt.Errorf("nil level provided")
	}

	// Convert collision polygons to BSP tree
	var bspPolygons []bsp.Polygon
	for _, collision := range yamlLevel.Collisions {
		vertices := make([]bsp.Point, len(collision.Outline))
		for i, v := range collision.Outline {
			vertices[i] = bsp.Point{X: v.X, Y: v.Y}
		}
		bspPolygons = append(bspPolygons, bsp.Polygon{
			Vertices: vertices,
			IsSolid:  true,
		})
	}

	// Build BSP tree
	builder := bsp.NewBSPBuilder(bspPolygons)
	bspRoot := builder.Build()

	// Convert ground tiles
	groundTiles := make([]*pb.Tile, len(yamlLevel.Ground))
	for i, tile := range yamlLevel.Ground {
		groundTiles[i] = &pb.Tile{
			Position: &pb.Vec2I{
				X: tile.Position.X,
				Y: tile.Position.Y,
			},
			Texture: tile.Texture,
		}
	}

	// Create level data
	levelData := &pb.LevelData{
		Root:   bspRoot,
		Ground: groundTiles,
	}

	return levelData, nil
}

// buildLevelsIterator creates an iterator that yields (relativePath, protoBytes) pairs
// for each level file, with a 30-second timeout per level conversion.
// If any level times out, the build fails with an error.
func buildLevelsIterator(assetsDir string) iter.Seq2[string, []byte] {
	return func(yield func(string, []byte) bool) {
		levelsDir := filepath.Join(assetsDir, "levels")

		// Find all YAML level files
		var matches []string
		err := filepath.Walk(levelsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".yaml") {
				matches = append(matches, path)
			}
			return nil
		})

		if err != nil && !os.IsNotExist(err) {
			fmt.Printf("Error walking levels directory: %v\n", err)
			return
		}

		// Process each level with timeout
		for _, yamlPath := range matches {
			// Create context with 30-second timeout
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

			// Channel to receive the result
			type result struct {
				relPath string
				bytes   []byte
				err     error
			}
			resultChan := make(chan result, 1)

			// Run conversion in goroutine
			go func() {
				defer cancel()

				// Load the YAML level
				lvl := level.New()
				if err := lvl.Load(yamlPath); err != nil {
					resultChan <- result{err: fmt.Errorf("loading level %s: %w", yamlPath, err)}
					return
				}

				// Convert to protobuf
				protoLevel, err := convertLevelToProto(lvl)
				if err != nil {
					resultChan <- result{err: fmt.Errorf("converting level %s to protobuf: %w", yamlPath, err)}
					return
				}

				// Serialize to bytes
				protoBytes, err := proto.Marshal(protoLevel)
				if err != nil {
					resultChan <- result{err: fmt.Errorf("marshaling level %s: %w", yamlPath, err)}
					return
				}

				// Get relative path from assets directory and change extension
				relPath, err := filepath.Rel(assetsDir, yamlPath)
				if err != nil {
					resultChan <- result{err: fmt.Errorf("getting relative path for %s: %w", yamlPath, err)}
					return
				}
				// Change .yaml extension to .pb
				relPath = strings.TrimSuffix(relPath, ".yaml") + ".pb"

				resultChan <- result{relPath: relPath, bytes: protoBytes, err: nil}
			}()

			// Wait for result or timeout
			select {
			case <-ctx.Done():
				// Timeout occurred
				fmt.Printf("ERROR: Level conversion timed out after 30s: %s\n", yamlPath)
				cancel()
				return // Stop iteration, build will fail
			case res := <-resultChan:
				cancel()
				if res.err != nil {
					fmt.Printf("ERROR: %v\n", res.err)
					return // Stop iteration, build will fail
				}

				// Yield the result
				fmt.Printf("  Converted: %s -> %s\n", filepath.Base(yamlPath), res.relPath)
				if !yield(res.relPath, res.bytes) {
					return // Consumer requested stop
				}
			}
		}
	}
}
