package clay

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Compile compiles clay_glue.c to an object file.
// Returns the path to the compiled object file.
// useZig determines whether to try using Zig for cross-compilation.
func Compile(clayDir, target string, useZig bool) (string, error) {
	claySource := filepath.Join(clayDir, "clay_glue.c")
	clayHeader := filepath.Join(clayDir, "clay.h")
	clayObject := filepath.Join(clayDir, "clay_glue.o")

	// Check if recompilation is needed
	needsRecompile, err := needsRecompile(claySource, clayHeader, clayObject)
	if err != nil {
		return "", fmt.Errorf("checking recompile status: %w", err)
	}

	if !needsRecompile {
		fmt.Println("Clay object file is up to date")
		return clayObject, nil
	}

	fmt.Println("Compiling clay_glue.c...")

	// Determine if we're cross-compiling
	currentPlatform := getCurrentPlatform()
	isCrossCompiling := !isTargetCompatible(currentPlatform, target)

	var compiler string
	var args []string

	if isCrossCompiling && useZig {
		// Try to use Zig for cross-compilation
		zigPath, err := exec.LookPath("zig")
		if err != nil {
			return "", fmt.Errorf("zig not found (required for cross-compilation). Please install Zig: https://ziglang.org/download/")
		}

		compiler = zigPath
		args = []string{"cc", "-c", claySource, "-o", clayObject, "-O2"}

		// Add target flag for Zig
		zigTarget := mapOdinToZigTarget(target)
		if zigTarget != "" {
			args = append(args, "-target", zigTarget)
		}
	} else {
		// Use clang for same-platform compilation
		clangPath, err := exec.LookPath("clang")
		if err != nil {
			return "", fmt.Errorf("clang not found. Please install clang compiler")
		}

		compiler = clangPath
		args = []string{"-c", claySource, "-o", clayObject, "-O2"}
	}

	cmd := exec.Command(compiler, args...)
	cmd.Dir = clayDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("compiling clay_glue.c: %w\nOutput: %s", err, string(output))
	}

	fmt.Println("Clay compiled successfully")
	return clayObject, nil
}

// needsRecompile checks if the object file needs to be recompiled.
func needsRecompile(source, header, object string) (bool, error) {
	objInfo, err := os.Stat(object)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("stat object file: %w", err)
	}

	srcInfo, err := os.Stat(source)
	if err != nil {
		return false, fmt.Errorf("stat source file: %w", err)
	}

	hdrInfo, err := os.Stat(header)
	if err != nil {
		return false, fmt.Errorf("stat header file: %w", err)
	}

	objTime := objInfo.ModTime()
	return objTime.Before(srcInfo.ModTime()) || objTime.Before(hdrInfo.ModTime()), nil
}

// getCurrentPlatform returns the current platform in a normalized format.
func getCurrentPlatform() string {
	return runtime.GOOS
}

// isTargetCompatible checks if the target is compatible with the current platform.
func isTargetCompatible(currentPlatform, target string) bool {
	switch currentPlatform {
	case "darwin":
		return target == "darwin_arm64" || target == "darwin_amd64"
	case "linux":
		return target == "linux_amd64" || target == "linux_i386"
	case "windows":
		return target == "windows_amd64" || target == "windows_i386"
	default:
		return false
	}
}

// mapOdinToZigTarget maps Odin target strings to Zig target triples.
func mapOdinToZigTarget(odinTarget string) string {
	switch odinTarget {
	case "darwin_arm64":
		return "aarch64-macos"
	case "darwin_amd64":
		return "x86_64-macos"
	case "linux_amd64":
		return "x86_64-linux"
	case "linux_i386":
		return "i386-linux"
	case "windows_amd64":
		return "x86_64-windows"
	case "windows_i386":
		return "i386-windows"
	default:
		return ""
	}
}

