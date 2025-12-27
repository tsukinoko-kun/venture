package steamworks

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	steamworksRepo   = "rlabrecque/SteamworksSDK"
	steamworksCommit = "e7bb839178fc5a48aa380d85e2ad04cc97d9d11c" // v1.60
)

// LibraryInfo contains information about Steam libraries for a target.
type LibraryInfo struct {
	RuntimeLib string // Path to runtime library (.dylib/.so/.dll)
	LinkLib    string // Path to link library (.lib for Windows, empty otherwise)
}

// EnsureLibraries ensures Steam libraries are downloaded for the target platform.
// Returns paths to runtime and link libraries (link lib is empty for non-Windows).
func EnsureLibraries(target, steamworksDir string) (*LibraryInfo, error) {
	runtimeInfo := getRuntimeLibraryInfo(target)
	if runtimeInfo == nil {
		return nil, fmt.Errorf("unsupported Steam platform: %s", target)
	}

	remotePath, subdir, filename := runtimeInfo.RemotePath, runtimeInfo.Subdir, runtimeInfo.Filename
	localPath := filepath.Join(steamworksDir, subdir, filename)

	if err := downloadFile(remotePath, localPath); err != nil {
		return nil, fmt.Errorf("downloading runtime library: %w", err)
	}

	info := &LibraryInfo{
		RuntimeLib: localPath,
	}

	// Windows also needs the .lib file for linking
	if strings.HasPrefix(target, "windows") {
		linkInfo := getLinkLibraryInfo(target)
		if linkInfo != nil {
			linkRemote, linkSubdir, linkFilename := linkInfo.RemotePath, linkInfo.Subdir, linkInfo.Filename
			linkLocal := filepath.Join(steamworksDir, linkSubdir, linkFilename)

			if err := downloadFile(linkRemote, linkLocal); err != nil {
				return nil, fmt.Errorf("downloading link library: %w", err)
			}

			info.LinkLib = linkLocal
		}
	}

	return info, nil
}

type libraryInfo struct {
	RemotePath string
	Subdir     string
	Filename   string
}

// getRuntimeLibraryInfo returns the remote path, local subdirectory, and filename for the Steam API runtime library.
func getRuntimeLibraryInfo(target string) *libraryInfo {
	switch {
	case strings.HasPrefix(target, "darwin"):
		return &libraryInfo{
			RemotePath: "redistributable_bin/osx/libsteam_api.dylib",
			Subdir:     "osx",
			Filename:   "libsteam_api.dylib",
		}
	case target == "linux_amd64":
		return &libraryInfo{
			RemotePath: "redistributable_bin/linux64/libsteam_api.so",
			Subdir:     "linux64",
			Filename:   "libsteam_api.so",
		}
	case strings.Contains(target, "arm64") || strings.Contains(target, "aarch64"):
		return &libraryInfo{
			RemotePath: "redistributable_bin/linuxarm64/libsteam_api.so",
			Subdir:     "linuxarm64",
			Filename:   "libsteam_api.so",
		}
	case target == "windows_amd64":
		return &libraryInfo{
			RemotePath: "redistributable_bin/win64/steam_api64.dll",
			Subdir:     "win64",
			Filename:   "steam_api64.dll",
		}
	}
	return nil
}

// getLinkLibraryInfo returns the link library info (for Windows .lib file).
func getLinkLibraryInfo(target string) *libraryInfo {
	if target == "windows_amd64" {
		return &libraryInfo{
			RemotePath: "redistributable_bin/win64/steam_api64.lib",
			Subdir:     "win64",
			Filename:   "steam_api64.lib",
		}
	}
	return nil
}

// downloadFile downloads a file from the Steamworks repo if not already present.
func downloadFile(remotePath, localPath string) error {
	// Check if file already exists
	if _, err := os.Stat(localPath); err == nil {
		return nil
	}

	// Create directory if needed
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s",
		steamworksRepo, steamworksCommit, remotePath)

	fmt.Printf("Downloading %s from GitHub...\n", filepath.Base(localPath))

	// Download file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	// Create file
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer out.Close()

	// Write content
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	// Make executable on Unix for .so and .dylib
	if strings.HasSuffix(localPath, ".so") || strings.HasSuffix(localPath, ".dylib") {
		if err := os.Chmod(localPath, 0755); err != nil {
			return fmt.Errorf("setting executable permission: %w", err)
		}
	}

	fmt.Printf("Downloaded %s\n", filepath.Base(localPath))
	return nil
}
