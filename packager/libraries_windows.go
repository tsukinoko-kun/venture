package packager

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DLLInfo contains information about a DLL to download.
type DLLInfo struct {
	Name        string // e.g., "SDL3"
	Version     string // e.g., "3.2.28"
	FileName    string // e.g., "SDL3.dll"
	ArchiveName string // e.g., "SDL3-3.2.28-win32-x64.zip"
	BaseURL     string // GitHub release URL base
}

// GetSDLDLLs returns the list of SDL DLLs to download based on version configuration.
func GetSDLDLLs(sdlVersion, sdlTTFVersion, sdlImageVersion string) []DLLInfo {
	dlls := []DLLInfo{}

	if sdlVersion != "" {
		dlls = append(dlls, DLLInfo{
			Name:        "SDL3",
			Version:     sdlVersion,
			FileName:    "SDL3.dll",
			ArchiveName: fmt.Sprintf("SDL3-%s-win32-x64.zip", sdlVersion),
			BaseURL:     "https://github.com/libsdl-org/SDL/releases/download",
		})
	}

	if sdlTTFVersion != "" {
		dlls = append(dlls, DLLInfo{
			Name:        "SDL3_ttf",
			Version:     sdlTTFVersion,
			FileName:    "SDL3_ttf.dll",
			ArchiveName: fmt.Sprintf("SDL3_ttf-%s-win32-x64.zip", sdlTTFVersion),
			BaseURL:     "https://github.com/libsdl-org/SDL_ttf/releases/download",
		})
	}

	if sdlImageVersion != "" {
		dlls = append(dlls, DLLInfo{
			Name:        "SDL3_image",
			Version:     sdlImageVersion,
			FileName:    "SDL3_image.dll",
			ArchiveName: fmt.Sprintf("SDL3_image-%s-win32-x64.zip", sdlImageVersion),
			BaseURL:     "https://github.com/libsdl-org/SDL_image/releases/download",
		})
	}

	return dlls
}

// getCacheDir returns the user's cache directory for DLLs.
func getCacheDir() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("getting user cache directory: %w", err)
	}

	cacheDir := filepath.Join(userCacheDir, "venture", "windows-dlls")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("creating cache directory: %w", err)
	}

	return cacheDir, nil
}

// getVersionedCacheDir returns the cache directory for a specific version.
func getVersionedCacheDir(cacheDir, name, version string) string {
	// Create a hash of the version to keep directory names clean
	hash := sha256.Sum256([]byte(name + "-" + version))
	versionHash := fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes of hash

	return filepath.Join(cacheDir, fmt.Sprintf("%s-%s-%s", name, version, versionHash))
}

// isDLLCached checks if a DLL is already cached.
func isDLLCached(cacheDir, name, version, fileName string) bool {
	versionedDir := getVersionedCacheDir(cacheDir, name, version)
	dllPath := filepath.Join(versionedDir, fileName)

	_, err := os.Stat(dllPath)
	return err == nil
}

// downloadAndExtractDLL downloads a DLL archive from GitHub and extracts the DLL.
func downloadAndExtractDLL(info DLLInfo, cacheDir string) error {
	versionedDir := getVersionedCacheDir(cacheDir, info.Name, info.Version)

	// Create versioned cache directory
	if err := os.MkdirAll(versionedDir, 0755); err != nil {
		return fmt.Errorf("creating versioned cache directory: %w", err)
	}

	// Construct download URL
	// Format: https://github.com/libsdl-org/SDL/releases/download/release-3.2.28/SDL3-3.2.28-win32-x64.zip
	releaseTag := fmt.Sprintf("release-%s", info.Version)
	downloadURL := fmt.Sprintf("%s/%s/%s", info.BaseURL, releaseTag, info.ArchiveName)

	fmt.Printf("  Downloading %s %s...\n", info.Name, info.Version)
	fmt.Printf("    URL: %s\n", downloadURL)

	// Download the archive
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", info.ArchiveName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Read the entire archive into memory
	archiveData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading archive data: %w", err)
	}

	fmt.Printf("    Downloaded %d bytes\n", len(archiveData))

	// Extract the DLL from the zip archive
	if err := extractDLLFromZip(archiveData, info.FileName, versionedDir); err != nil {
		return fmt.Errorf("extracting DLL: %w", err)
	}

	fmt.Printf("    ✅ Extracted %s to cache\n", info.FileName)
	return nil
}

// extractDLLFromZip extracts a specific DLL file from a zip archive in memory.
func extractDLLFromZip(archiveData []byte, dllFileName, targetDir string) error {
	// Create a reader from the archive data
	reader := bytes.NewReader(archiveData)
	zipReader, err := zip.NewReader(reader, int64(len(archiveData)))
	if err != nil {
		return fmt.Errorf("opening zip archive: %w", err)
	}

	// Find and extract the DLL file
	found := false
	for _, file := range zipReader.File {
		// Look for the DLL in any subdirectory (usually in a bin/ or lib/ folder)
		if strings.HasSuffix(strings.ToLower(file.Name), strings.ToLower(dllFileName)) {
			found = true

			// Open the file from the zip
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("opening %s in archive: %w", file.Name, err)
			}
			defer rc.Close()

			// Create the target file
			targetPath := filepath.Join(targetDir, dllFileName)
			targetFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("creating target file: %w", err)
			}
			defer targetFile.Close()

			// Copy the DLL
			if _, err := io.Copy(targetFile, rc); err != nil {
				return fmt.Errorf("extracting %s: %w", dllFileName, err)
			}

			break
		}
	}

	if !found {
		return fmt.Errorf("DLL file %s not found in archive", dllFileName)
	}

	return nil
}

// EnsureDLLsDownloaded ensures all required DLLs are downloaded and cached.
// Returns a map of DLL file names to their cached paths.
func EnsureDLLsDownloaded(dlls []DLLInfo) (map[string]string, error) {
	if len(dlls) == 0 {
		return make(map[string]string), nil
	}

	fmt.Println("Checking Windows DLL dependencies...")

	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, err
	}

	dllPaths := make(map[string]string)

	for _, dll := range dlls {
		versionedDir := getVersionedCacheDir(cacheDir, dll.Name, dll.Version)
		dllPath := filepath.Join(versionedDir, dll.FileName)

		if isDLLCached(cacheDir, dll.Name, dll.Version, dll.FileName) {
			fmt.Printf("  ✓ %s %s (cached)\n", dll.Name, dll.Version)
			dllPaths[dll.FileName] = dllPath
			continue
		}

		// Download and extract the DLL
		if err := downloadAndExtractDLL(dll, cacheDir); err != nil {
			return nil, fmt.Errorf("downloading %s: %w", dll.Name, err)
		}

		dllPaths[dll.FileName] = dllPath
	}

	return dllPaths, nil
}

// CopyDLLsToPackage copies all cached DLLs to the package directory.
func CopyDLLsToPackage(dllPaths map[string]string, packageDir string) error {
	if len(dllPaths) == 0 {
		return nil
	}

	fmt.Println("Copying DLLs to package...")

	for fileName, sourcePath := range dllPaths {
		targetPath := filepath.Join(packageDir, fileName)
		if err := copyFile(sourcePath, targetPath); err != nil {
			return fmt.Errorf("copying %s: %w", fileName, err)
		}
		fmt.Printf("  Copied: %s\n", fileName)
	}

	return nil
}
