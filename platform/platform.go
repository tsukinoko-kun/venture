package platform

import (
	"fmt"
	"runtime"
	"strings"
)

// DetectCurrent returns the current platform as an Odin target string.
func DetectCurrent() (string, error) {
	system := runtime.GOOS
	arch := runtime.GOARCH

	switch system {
	case "darwin":
		if arch == "arm64" {
			return "darwin_arm64", nil
		}
		return "darwin_amd64", nil
	case "linux":
		if arch == "amd64" {
			return "linux_amd64", nil
		}
		return "linux_i386", nil
	case "windows":
		if arch == "amd64" {
			return "windows_amd64", nil
		}
		return "windows_i386", nil
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", system, arch)
	}
}

// GetOutputName returns the output binary name for the given target.
func GetOutputName(target, binaryName string) string {
	if strings.HasPrefix(target, "windows") {
		return binaryName + ".exe"
	}
	return binaryName
}
