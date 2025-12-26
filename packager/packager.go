package packager

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PackageConfig holds the configuration for packaging.
type PackageConfig struct {
	BinaryPath string   // Path to the compiled binary
	BinaryName string   // Name of the binary (without extension)
	AssetsDir  string   // Path to the assets directory
	Libraries  []string // Paths to dynamic libraries to include
	Target     string   // Target platform (e.g., "darwin_arm64")
	OutputDir  string   // Directory to output the zip file
}

// Package creates a distribution zip file with the binary, assets, and libraries.
// Returns the path to the created zip file.
func Package(config PackageConfig) (string, error) {
	fmt.Println("Packaging for distribution...")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	// Determine zip filename
	zipName := fmt.Sprintf("%s-%s.zip", config.BinaryName, config.Target)
	zipPath := filepath.Join(config.OutputDir, zipName)

	// Create zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("creating zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add binary
	if err := addFileToZip(zipWriter, config.BinaryPath, filepath.Base(config.BinaryPath)); err != nil {
		return "", fmt.Errorf("adding binary to zip: %w", err)
	}

	// Add assets directory (preserving structure)
	if err := addDirToZip(zipWriter, config.AssetsDir, "assets"); err != nil {
		return "", fmt.Errorf("adding assets to zip: %w", err)
	}

	// Add libraries
	for _, libPath := range config.Libraries {
		if err := addFileToZip(zipWriter, libPath, filepath.Base(libPath)); err != nil {
			return "", fmt.Errorf("adding library %s to zip: %w", filepath.Base(libPath), err)
		}
	}

	fmt.Printf("✅ Package created: %s\n", zipPath)
	return zipPath, nil
}

// addFileToZip adds a single file to the zip archive.
func addFileToZip(zipWriter *zip.Writer, filePath, nameInZip string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("getting file info: %w", err)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("creating zip header: %w", err)
	}

	header.Name = nameInZip
	header.Method = zip.Deflate

	// Preserve executable permissions
	if info.Mode()&0111 != 0 {
		header.SetMode(0755)
	}

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("creating zip entry: %w", err)
	}

	if _, err := io.Copy(writer, file); err != nil {
		return fmt.Errorf("writing file to zip: %w", err)
	}

	fmt.Printf("  Added: %s\n", nameInZip)
	return nil
}

// addDirToZip adds a directory and its contents to the zip archive recursively.
func addDirToZip(zipWriter *zip.Writer, dirPath, nameInZip string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == dirPath {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return fmt.Errorf("calculating relative path: %w", err)
		}

		// Create path in zip (use forward slashes)
		zipPath := filepath.Join(nameInZip, relPath)
		zipPath = filepath.ToSlash(zipPath)

		if info.IsDir() {
			// Create directory entry
			header := &zip.FileHeader{
				Name:   zipPath + "/",
				Method: zip.Deflate,
			}
			if _, err := zipWriter.CreateHeader(header); err != nil {
				return fmt.Errorf("creating directory entry: %w", err)
			}
		} else {
			// Add file
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("opening file: %w", err)
			}
			defer file.Close()

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return fmt.Errorf("creating zip header: %w", err)
			}

			header.Name = zipPath
			header.Method = zip.Deflate

			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				return fmt.Errorf("creating zip entry: %w", err)
			}

			if _, err := io.Copy(writer, file); err != nil {
				return fmt.Errorf("writing file to zip: %w", err)
			}

			// Only print files, not directories
			if !strings.HasSuffix(zipPath, "/") {
				fmt.Printf("  Added: %s\n", zipPath)
			}
		}

		return nil
	})
}

