package protobuf

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

	// Save hash cache for future comparison
	if err := saveHashCache(protoFiles, outputDir); err != nil {
		fmt.Printf("Warning: Failed to save hash cache: %v\n", err)
	}

	return nil
}

const hashCacheFile = ".protobuf-hashes.json"

// protoHashCache stores hashes of proto files to detect changes.
type protoHashCache struct {
	Hashes map[string]string `json:"hashes"`
}

// needsRegeneration checks if any proto file has changed by comparing file hashes.
func needsRegeneration(protoFiles []string, outputDir string) (bool, error) {
	// Check if any generated files exist
	generatedFiles, err := filepath.Glob(filepath.Join(outputDir, "*.odin"))
	if err != nil {
		return false, fmt.Errorf("finding generated files: %w", err)
	}
	if len(generatedFiles) == 0 {
		return true, nil
	}

	// Load cached hashes
	cachePath := filepath.Join(outputDir, hashCacheFile)
	cached, err := loadHashCache(cachePath)
	if err != nil {
		// Cache doesn't exist or is invalid, regenerate
		return true, nil
	}

	// Compute current hashes and compare
	currentHashes, err := computeProtoHashes(protoFiles)
	if err != nil {
		return false, fmt.Errorf("computing hashes: %w", err)
	}

	// Check if the set of files changed
	if len(currentHashes) != len(cached.Hashes) {
		return true, nil
	}

	// Compare each hash
	for file, hash := range currentHashes {
		if cached.Hashes[file] != hash {
			return true, nil
		}
	}

	return false, nil
}

// saveHashCache saves the hash cache after successful generation.
func saveHashCache(protoFiles []string, outputDir string) error {
	hashes, err := computeProtoHashes(protoFiles)
	if err != nil {
		return fmt.Errorf("computing hashes: %w", err)
	}

	cache := protoHashCache{Hashes: hashes}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}

	cachePath := filepath.Join(outputDir, hashCacheFile)
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("writing cache: %w", err)
	}

	return nil
}

// loadHashCache loads the hash cache from disk.
func loadHashCache(cachePath string) (*protoHashCache, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cache protoHashCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// computeProtoHashes computes SHA256 hashes for all proto files.
func computeProtoHashes(protoFiles []string) (map[string]string, error) {
	hashes := make(map[string]string)

	// Sort files for consistent ordering
	sorted := make([]string, len(protoFiles))
	copy(sorted, protoFiles)
	sort.Strings(sorted)

	for _, protoFile := range sorted {
		hash, err := hashFile(protoFile)
		if err != nil {
			return nil, fmt.Errorf("hashing %s: %w", protoFile, err)
		}
		// Use base name as key for portability across machines
		hashes[filepath.Base(protoFile)] = hash
	}

	return hashes, nil
}

// hashFile computes the SHA256 hash of a file.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
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

// Clean removes generated protobuf files and the hash cache.
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

	// Remove hash cache
	cachePath := filepath.Join(outputDir, hashCacheFile)
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing hash cache: %w", err)
	}

	return nil
}
