package packager

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LibraryVersions contains version information for SDL libraries.
// Used to download platform-specific binaries from official releases.
type LibraryVersions struct {
	SDL      string // SDL3 version (e.g., "3.2.28")
	SDLTTF   string // SDL3_ttf version (e.g., "3.2.2")
	SDLImage string // SDL3_image version (e.g., "3.2.4")
}

// PackageConfig holds the configuration for packaging.
type PackageConfig struct {
	ProjectRoot     string          // Root directory of the project
	BinaryPath      string          // Path to the compiled binary
	BinaryName      string          // Name of the binary (without extension)
	AssetsDir       string          // Path to the assets directory
	Libraries       []string        // Paths to dynamic libraries to include (e.g., Steam)
	LibraryVersions LibraryVersions // Versions of SDL libraries to download
	Target          string          // Target platform (e.g., "darwin_arm64")
	OutputDir       string          // Directory to output the package
}

// Package creates a distribution package with the binary, assets, and libraries.
// On macOS: uses dylibbundler to bundle shared libraries, creates zip
// On Linux: uses linuxdeploy to bundle dependencies, creates zip with binary and lib/ folder
// On Windows: bundles DLLs and creates zip
// Returns the path to the created package.
func Package(config PackageConfig) (string, error) {
	fmt.Println("Packaging for distribution...")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	// Platform-specific packaging is implemented in:
	// - packager_darwin.go (macOS)
	// - packager_linux.go (Linux)
	// - packager_windows.go (Windows)
	// These files use build tags so only the relevant code compiles per platform
	return packagePlatform(config)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Preserve permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .DS_Store and other hidden files
		if info.Name() == ".DS_Store" || strings.HasPrefix(info.Name(), "._") {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath)
	})
}

// createZipArchive creates a zip file from a directory
func createZipArchive(sourceDir, zipPath, baseName string) error {
	// Create zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("creating zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk through the source directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .DS_Store and other hidden files
		if info.Name() == ".DS_Store" || strings.HasPrefix(info.Name(), "._") {
			return nil
		}

		// Calculate relative path from source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Create path in zip with baseName prefix
		zipPath := filepath.Join(baseName, relPath)
		zipPath = filepath.ToSlash(zipPath) // Use forward slashes in zip

		if info.IsDir() {
			// Create directory entry
			header := &zip.FileHeader{
				Name:   zipPath + "/",
				Method: zip.Deflate,
			}
			_, err := zipWriter.CreateHeader(header)
			return err
		}

		// Add file
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = zipPath
		header.Method = zip.Deflate

		// Preserve executable permissions
		if info.Mode()&0111 != 0 {
			header.SetMode(0755)
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		_, err = io.Copy(writer, file)
		return err
	})
}
