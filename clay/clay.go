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

	// Try to find a C compiler (prefer clang, fallback to gcc, then cl on Windows)
	var compilerPath string
	var args []string

	// Try clang first
	clangPath, clangErr := exec.LookPath("clang")
	if clangErr == nil {
		compilerPath = clangPath
		args = []string{"-c", claySource, "-o", clayObject, "-O2"}
	} else {
		// Try gcc
		gccPath, gccErr := exec.LookPath("gcc")
		if gccErr == nil {
			compilerPath = gccPath
			args = []string{"-c", claySource, "-o", clayObject, "-O2"}
		} else {
			// Try MSVC cl.exe on Windows
			clPath, clErr := exec.LookPath("cl")
			if clErr == nil {
				compilerPath = clPath
				// MSVC uses different flags: /c for compile only, /Fo for output
				args = []string{"/c", claySource, "/Fo" + clayObject, "/O2"}
			} else {
				return "", fmt.Errorf("no C compiler found. Tried clang (%v), gcc (%v), cl (%v)", clangErr, gccErr, clErr)
			}
		}
	}

	fmt.Printf("Using compiler: %s\n", compilerPath)
	cmd := exec.Command(compilerPath, args...)
	cmd.Dir = clayDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("compiling clay_glue.c with %s: %w", compilerPath, err)
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
