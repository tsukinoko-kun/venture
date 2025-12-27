package clay

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Compile compiles clay_glue.c to an object file.
// Returns the path to the compiled object file.
func Compile(clayDir, target string) (string, error) {
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

	// Use clang for compilation
	clangPath, err := exec.LookPath("clang")
	if err != nil {
		return "", fmt.Errorf("clang not found. Please install clang compiler")
	}

	args := []string{"-c", claySource, "-o", clayObject, "-O2"}

	cmd := exec.Command(clangPath, args...)
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
