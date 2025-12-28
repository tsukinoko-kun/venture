package packager

import (
	"fmt"
	"os"
	"path/filepath"
)

// packagePlatform creates a distribution package for Windows with all DLLs
func packagePlatform(config PackageConfig) (string, error) {
	fmt.Println("Creating Windows distribution package...")

	// Create temporary directory for packaging
	packageName := fmt.Sprintf("%s-%s", config.BinaryName, config.Target)
	packageDir := filepath.Join(config.OutputDir, packageName)

	// Remove old package directory if it exists
	if err := os.RemoveAll(packageDir); err != nil {
		return "", fmt.Errorf("removing old package directory: %w", err)
	}

	// Create package directory
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return "", fmt.Errorf("creating package directory: %w", err)
	}

	// Copy binary to package directory
	targetBinary := filepath.Join(packageDir, filepath.Base(config.BinaryPath))
	if err := copyFile(config.BinaryPath, targetBinary); err != nil {
		return "", fmt.Errorf("copying binary: %w", err)
	}
	fmt.Printf("  Copied binary: %s\n", filepath.Base(targetBinary))

	// Copy assets to package directory
	if _, err := os.Stat(config.AssetsDir); err == nil {
		assetsTarget := filepath.Join(packageDir, "assets")
		if err := copyDir(config.AssetsDir, assetsTarget); err != nil {
			return "", fmt.Errorf("copying assets: %w", err)
		}
		fmt.Printf("  Copied assets\n")
	}

	// Copy explicitly listed libraries (e.g., Steam DLLs)
	for _, libPath := range config.Libraries {
		libName := filepath.Base(libPath)
		targetLib := filepath.Join(packageDir, libName)
		if err := copyFile(libPath, targetLib); err != nil {
			return "", fmt.Errorf("copying library %s: %w", libName, err)
		}
		fmt.Printf("  Copied library: %s\n", libName)
	}

	// On Windows, we need to find and copy all DLL dependencies
	// We'll use a simple approach: look for common runtime DLLs in system paths
	fmt.Println("Scanning for DLL dependencies...")
	if err := findAndCopyWindowsDLLs(targetBinary, packageDir); err != nil {
		// Don't fail the build if DLL detection has issues, just warn
		fmt.Printf("Warning: Could not automatically detect all DLLs: %v\n", err)
		fmt.Println("You may need to manually include missing DLLs")
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

	fmt.Printf("\n✅ Windows package created: %s\n", zipPath)
	return zipPath, nil
}

// findAndCopyWindowsDLLs attempts to find and copy DLL dependencies for Windows
func findAndCopyWindowsDLLs(binaryPath, targetDir string) error {
	// Check if we're running on Windows and have access to dependency tools
	// For now, we'll use a simple heuristic: look in common locations

	// Common DLL locations on Windows
	searchPaths := []string{
		filepath.Dir(binaryPath), // Same directory as binary
		"C:\\Windows\\System32",
		"C:\\Windows\\SysWOW64",
	}

	// Common runtime DLLs that games might need (excluding system DLLs)
	// In practice, most DLLs will be found next to the binary or in vendor folders
	commonDLLs := []string{
		"SDL3.dll",
		"openal32.dll",
		"OpenAL32.dll",
		"libvorbis.dll",
		"libvorbisfile.dll",
		"libogg.dll",
	}

	copiedCount := 0
	for _, dllName := range commonDLLs {
		found := false
		for _, searchPath := range searchPaths {
			dllPath := filepath.Join(searchPath, dllName)
			if _, err := os.Stat(dllPath); err == nil {
				targetPath := filepath.Join(targetDir, dllName)
				// Don't overwrite if already exists
				if _, err := os.Stat(targetPath); err == nil {
					found = true
					break
				}
				if err := copyFile(dllPath, targetPath); err == nil {
					fmt.Printf("  Copied DLL: %s\n", dllName)
					copiedCount++
					found = true
					break
				}
			}
		}
		if !found {
			// Not an error - the binary might not need this DLL
		}
	}

	if copiedCount > 0 {
		fmt.Printf("  Found and copied %d additional DLL(s)\n", copiedCount)
	} else {
		fmt.Println("  No additional DLLs found (this is normal if all dependencies are statically linked)")
	}

	return nil
}
