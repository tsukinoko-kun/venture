package packager

import (
	"fmt"
	"os"
	"path/filepath"
)

// packagePlatform creates a distribution package for Windows with all DLLs
func packagePlatform(config PackageConfig) (string, error) {
	fmt.Println("Creating Windows distribution package...")

	// Step 1: Download SDL DLLs if library versions are specified
	dlls := GetSDLDLLs(
		config.LibraryVersions.SDL,
		config.LibraryVersions.SDLTTF,
		config.LibraryVersions.SDLImage,
	)

	dllPaths, err := EnsureDLLsDownloaded(dlls)
	if err != nil {
		return "", fmt.Errorf("downloading DLLs: %w", err)
	}

	// Create temporary directory for packaging (outside of build dir)
	packageName := fmt.Sprintf("%s-%s", config.BinaryName, config.Target)
	tempDir, err := os.MkdirTemp("", packageName+"-*")
	if err != nil {
		return "", fmt.Errorf("creating temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up temp directory when done

	packageDir := filepath.Join(tempDir, packageName)

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

	// Copy downloaded SDL DLLs to package directory
	if err := CopyDLLsToPackage(dllPaths, packageDir); err != nil {
		return "", fmt.Errorf("copying SDL DLLs: %w", err)
	}

	// Create zip archive in the build directory
	zipName := packageName + ".zip"
	zipPath := filepath.Join(config.OutputDir, zipName)

	// Ensure output directory exists
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	fmt.Println("Creating zip archive...")
	if err := createZipArchive(packageDir, zipPath, packageName); err != nil {
		return "", fmt.Errorf("creating zip archive: %w", err)
	}

	fmt.Printf("\n✅ Windows package created: %s\n", zipPath)
	return zipPath, nil
}
