package packager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// packagePlatform creates a distribution package using dylibbundler and zips it
func packagePlatform(config PackageConfig) (string, error) {
	fmt.Println("Creating macOS distribution package...")

	// Check if dylibbundler is available
	if _, err := exec.LookPath("dylibbundler"); err != nil {
		return "", fmt.Errorf("dylibbundler not found. Install it with: brew install dylibbundler")
	}

	// Create temporary directory for packaging
	packageName := fmt.Sprintf("%s-%s", config.BinaryName, config.Target)
	packageDir := filepath.Join(config.OutputDir, packageName)

	// Remove old package directory if it exists
	if err := os.RemoveAll(packageDir); err != nil {
		return "", fmt.Errorf("removing old package directory: %w", err)
	}

	// Create package directory and libs subdirectory
	libsDir := filepath.Join(packageDir, "libs")
	if err := os.MkdirAll(libsDir, 0755); err != nil {
		return "", fmt.Errorf("creating package directory: %w", err)
	}

	// Copy binary to package directory
	targetBinary := filepath.Join(packageDir, config.BinaryName)
	if err := copyFile(config.BinaryPath, targetBinary); err != nil {
		return "", fmt.Errorf("copying binary: %w", err)
	}

	// Make binary executable
	if err := os.Chmod(targetBinary, 0755); err != nil {
		return "", fmt.Errorf("making binary executable: %w", err)
	}
	fmt.Printf("  Copied binary to %s\n", targetBinary)

	// Copy assets to package directory
	if _, err := os.Stat(config.AssetsDir); err == nil {
		assetsTarget := filepath.Join(packageDir, "assets")
		if err := copyDir(config.AssetsDir, assetsTarget); err != nil {
			return "", fmt.Errorf("copying assets: %w", err)
		}
		fmt.Printf("  Copied assets\n")
	}

	// Copy any explicitly listed libraries (e.g., Steam)
	for _, libPath := range config.Libraries {
		libName := filepath.Base(libPath)
		targetLib := filepath.Join(libsDir, libName)
		if err := copyFile(libPath, targetLib); err != nil {
			return "", fmt.Errorf("copying library %s: %w", libName, err)
		}
		fmt.Printf("  Copied library: %s\n", libName)
	}

	// Run dylibbundler to bundle all shared libraries
	fmt.Println("Running dylibbundler to bundle shared libraries...")
	cmd := exec.Command("dylibbundler",
		"-od",              // Overwrite files
		"-b",               // Bundle dependencies
		"-x", targetBinary, // Executable
		"-d", libsDir, // Destination for libraries
		"-p", "@executable_path/libs/", // Install path relative to binary
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// dylibbundler can be noisy but still succeed, check if there are actual errors
		fmt.Printf("dylibbundler output: %s\n", string(output))
		if !strings.Contains(string(output), "Error") {
			fmt.Println("  ✅ Shared libraries bundled (with warnings)")
		} else {
			return "", fmt.Errorf("running dylibbundler: %w\nOutput: %s", err, string(output))
		}
	} else {
		fmt.Println("  ✅ Shared libraries bundled successfully")
	}

	// Create zip archive
	zipName := packageName + ".zip"
	zipPath := filepath.Join(config.OutputDir, zipName)

	fmt.Println("Creating zip archive...")
	if err := createZipArchive(packageDir, zipPath, packageName); err != nil {
		return "", fmt.Errorf("creating zip archive: %w", err)
	}

	// Clean up temporary package directory
	if err := os.RemoveAll(packageDir); err != nil {
		fmt.Printf("Warning: Failed to clean up temporary directory: %v\n", err)
	}

	fmt.Printf("\n✅ macOS package created: %s\n", zipPath)
	return zipPath, nil
}
