package platform

import (
	"fmt"
	"runtime"
	"strings"
)

// Odin target mappings
var odinTargets = map[string]string{
	"windows":     "windows_amd64",
	"linux":       "linux_amd64",
	"macos":       "darwin_arm64",
	"macos-intel": "darwin_amd64",
}

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

// MapTarget maps a friendly target name to an Odin target string.
// If the input is already an Odin target, it returns it unchanged.
func MapTarget(friendlyName string) string {
	if mapped, ok := odinTargets[friendlyName]; ok {
		return mapped
	}
	// If not in map, assume it's already an Odin target
	return friendlyName
}

// GetOutputName returns the output binary name for the given target.
func GetOutputName(target, binaryName string) string {
	if strings.HasPrefix(target, "windows") {
		return binaryName + ".exe"
	}
	return binaryName
}

// GetAllTargets returns a list of all valid target names (both friendly and Odin format).
func GetAllTargets() []string {
	targets := make([]string, 0, len(odinTargets)*2)

	// Add friendly names
	for friendly := range odinTargets {
		targets = append(targets, friendly)
	}

	// Add Odin format targets
	for _, odin := range odinTargets {
		targets = append(targets, odin)
	}

	return targets
}
