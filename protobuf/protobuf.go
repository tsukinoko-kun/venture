package protobuf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Generate generates Odin code from .proto files using protoc and odin-protoc-plugin.
func Generate(protoDir, outputDir string) error {
	// Find all .proto files
	protoFiles, err := filepath.Glob(filepath.Join(protoDir, "*.proto"))
	if err != nil {
		return fmt.Errorf("finding proto files: %w", err)
	}

	if len(protoFiles) == 0 {
		fmt.Println("No .proto files found, skipping protobuf generation")
		return nil
	}

	// Check if regeneration is needed by comparing timestamps
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	needsRegeneration, err := needsRegeneration(protoFiles, outputDir)
	if err != nil {
		return fmt.Errorf("checking timestamps: %w", err)
	}

	if !needsRegeneration {
		fmt.Println("Generated protobuf code is up to date")
		return nil
	}

	// Check if protoc is available
	if err := checkProtoc(); err != nil {
		return err
	}

	// Check if odin-protoc-plugin is available
	if err := checkOdinProtocPlugin(); err != nil {
		return err
	}

	fmt.Println("Generating Odin code from .proto files...")

	// Run protoc for each proto file
	for _, protoFile := range protoFiles {
		cmd := exec.Command(
			"protoc",
			fmt.Sprintf("--proto_path=%s", protoDir),
			fmt.Sprintf("--odin_out=%s", outputDir),
			protoFile,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("generating code for %s: %w\nOutput: %s",
				filepath.Base(protoFile), err, string(output))
		}

		fmt.Printf("  Generated code from %s\n", filepath.Base(protoFile))
	}

	fmt.Println("Protobuf code generation completed")

	// Format generated code
	fmt.Println("Formatting generated code...")
	cmd := exec.Command("odinfmt", "-w", outputDir)
	if err := cmd.Run(); err != nil {
		// Don't fail the build if formatting fails
		fmt.Printf("Warning: Failed to format generated code: %v\n", err)
	} else {
		fmt.Println("Formatting completed")
	}

	return nil
}

// needsRegeneration checks if any proto file is newer than the generated files.
func needsRegeneration(protoFiles []string, outputDir string) (bool, error) {
	// Find newest proto file
	var newestProto int64
	for _, protoFile := range protoFiles {
		info, err := os.Stat(protoFile)
		if err != nil {
			return false, fmt.Errorf("stat proto file: %w", err)
		}
		if info.ModTime().Unix() > newestProto {
			newestProto = info.ModTime().Unix()
		}
	}

	// Find oldest generated file
	generatedFiles, err := filepath.Glob(filepath.Join(outputDir, "*.odin"))
	if err != nil {
		return false, fmt.Errorf("finding generated files: %w", err)
	}

	if len(generatedFiles) == 0 {
		return true, nil
	}

	var oldestGenerated int64 = 9999999999
	for _, genFile := range generatedFiles {
		info, err := os.Stat(genFile)
		if err != nil {
			return false, fmt.Errorf("stat generated file: %w", err)
		}
		if info.ModTime().Unix() < oldestGenerated {
			oldestGenerated = info.ModTime().Unix()
		}
	}

	return oldestGenerated <= newestProto, nil
}

// checkProtoc checks if protoc is available.
func checkProtoc() error {
	cmd := exec.Command("protoc", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("protoc not found. Please install Protocol Buffers compiler.\n" +
			"  macOS: brew install protobuf\n" +
			"  Linux: apt install protobuf-compiler")
	}
	return nil
}

// checkOdinProtocPlugin checks if odin-protoc-plugin is available.
func checkOdinProtocPlugin() error {
	cmd := exec.Command("protoc-gen-odin", "--version")
	if err := cmd.Run(); err != nil {
		// Check if it's in PATH
		_, pathErr := exec.LookPath("protoc-gen-odin")
		if pathErr != nil {
			return fmt.Errorf("protoc-gen-odin not found. Please install the Odin protoc plugin.\n" +
				"  Download from: https://github.com/lordhippo/odin-protoc-plugin/releases\n" +
				"  Extract and place protoc-gen-odin in your PATH (e.g., ~/go/bin/ or /usr/local/bin/)")
		}
	}
	return nil
}

// Clean removes generated protobuf files.
func Clean(outputDir string) error {
	generatedFiles, err := filepath.Glob(filepath.Join(outputDir, "*.pb.odin"))
	if err != nil {
		return fmt.Errorf("finding generated files: %w", err)
	}

	for _, file := range generatedFiles {
		if strings.HasSuffix(file, ".pb.odin") {
			if err := os.Remove(file); err != nil {
				return fmt.Errorf("removing %s: %w", file, err)
			}
		}
	}

	return nil
}
