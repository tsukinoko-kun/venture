package packager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// packagePlatform creates a Linux distribution bundle with the binary and libraries
func packagePlatform(config PackageConfig) (string, error) {
	fmt.Println("Creating Linux distribution bundle...")

	// Check if linuxdeploy is available
	linuxdeployPath, err := exec.LookPath("linuxdeploy")
	if err != nil {
		return "", fmt.Errorf("linuxdeploy not found. Download it from: https://github.com/linuxdeploy/linuxdeploy/releases")
	}

	// Create temporary directory for packaging (outside of build dir)
	tempDir, err := os.MkdirTemp("", config.BinaryName+"-*")
	if err != nil {
		return "", fmt.Errorf("creating temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up temp directory when done

	// Create AppDir structure (we'll use linuxdeploy to bundle dependencies)
	appDirName := config.BinaryName + ".AppDir"
	appDir := filepath.Join(tempDir, appDirName)
	usrBinDir := filepath.Join(appDir, "usr", "bin")
	usrLibDir := filepath.Join(appDir, "usr", "lib")
	usrShareDir := filepath.Join(appDir, "usr", "share")

	for _, dir := range []string{usrBinDir, usrLibDir, usrShareDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	// Copy binary
	targetBinary := filepath.Join(usrBinDir, config.BinaryName)
	if err := copyFile(config.BinaryPath, targetBinary); err != nil {
		return "", fmt.Errorf("copying binary: %w", err)
	}
	if err := os.Chmod(targetBinary, 0755); err != nil {
		return "", fmt.Errorf("making binary executable: %w", err)
	}
	fmt.Printf("  Copied binary to %s\n", targetBinary)

	// Copy assets (excluding YAML level files)
	if _, err := os.Stat(config.AssetsDir); err == nil {
		assetsTarget := filepath.Join(usrShareDir, "assets")
		if err := copyDirExcludingLevels(config.AssetsDir, assetsTarget); err != nil {
			return "", fmt.Errorf("copying assets: %w", err)
		}
		fmt.Printf("  Copied assets (excluding YAML level files)\n")
	}

	// Write protobuf level files to AppDir from iterator
	if config.LevelIterator != nil {
		assetsTarget := filepath.Join(usrShareDir, "assets")
		levelCount := 0
		for relPath, protoBytes := range config.LevelIterator {
			targetPath := filepath.Join(assetsTarget, relPath)

			// Ensure directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return "", fmt.Errorf("creating directory for level %s: %w", relPath, err)
			}

			// Write protobuf bytes immediately
			if err := os.WriteFile(targetPath, protoBytes, 0644); err != nil {
				return "", fmt.Errorf("writing level file %s: %w", relPath, err)
			}
			levelCount++
		}
		fmt.Printf("  Wrote %d protobuf level file(s)\n", levelCount)
	}

	// Copy any explicitly listed libraries
	for _, libPath := range config.Libraries {
		libName := filepath.Base(libPath)
		targetLib := filepath.Join(usrLibDir, libName)
		if err := copyFile(libPath, targetLib); err != nil {
			return "", fmt.Errorf("copying library %s: %w", libName, err)
		}
		fmt.Printf("  Copied library: %s\n", libName)
	}

	// Create .desktop file (required by linuxdeploy)
	desktopFile := filepath.Join(appDir, config.BinaryName+".desktop")
	desktopContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s
Icon=%s
Categories=Game;
Terminal=false
`, config.BinaryName, config.BinaryName, config.BinaryName)

	if err := os.WriteFile(desktopFile, []byte(desktopContent), 0644); err != nil {
		return "", fmt.Errorf("writing .desktop file: %w", err)
	}
	fmt.Println("  Created .desktop file")

	// Create a simple icon file (linuxdeploy requires one)
	iconFile := filepath.Join(appDir, config.BinaryName+".png")
	// Create a minimal 1x1 PNG as placeholder
	minimalPNG := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
		0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(iconFile, minimalPNG, 0644); err != nil {
		return "", fmt.Errorf("writing icon file: %w", err)
	}

	// Run linuxdeploy to bundle dependencies (but don't create AppImage)
	fmt.Println("Running linuxdeploy to bundle dependencies...")

	cmd := exec.Command(linuxdeployPath,
		"--appdir", appDir,
		"--executable", targetBinary,
	)
	cmd.Dir = tempDir

	output, err := cmd.CombinedOutput()
	fmt.Printf("linuxdeploy output:\n%s\n", string(output))

	if err != nil {
		return "", fmt.Errorf("running linuxdeploy: %w\nOutput: %s", err, string(output))
	}

	// Now create the final distribution structure
	distDirName := fmt.Sprintf("%s-%s", config.BinaryName, config.Target)
	distDir := filepath.Join(tempDir, distDirName)

	// Create dist directory structure
	distLibDir := filepath.Join(distDir, "lib")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return "", fmt.Errorf("creating distribution directory: %w", err)
	}
	if err := os.MkdirAll(distLibDir, 0755); err != nil {
		return "", fmt.Errorf("creating lib directory: %w", err)
	}

	// Copy the binary from AppDir/usr/bin to dist root
	finalBinary := filepath.Join(distDir, config.BinaryName)
	if err := copyFile(targetBinary, finalBinary); err != nil {
		return "", fmt.Errorf("copying binary to dist: %w", err)
	}
	if err := os.Chmod(finalBinary, 0755); err != nil {
		return "", fmt.Errorf("making binary executable: %w", err)
	}
	fmt.Printf("  Copied binary to distribution directory\n")

	// Ensure RPATH is set correctly for the binary to find libs in ./lib
	// Use patchelf if available to set RPATH to $ORIGIN/lib
	if patchelfPath, err := exec.LookPath("patchelf"); err == nil {
		fmt.Println("Setting RPATH with patchelf...")
		cmd := exec.Command(patchelfPath,
			"--set-rpath", "$ORIGIN/lib",
			finalBinary,
		)
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Warning: patchelf failed (binary may already have correct RPATH): %v\n%s\n", err, string(output))
		} else {
			fmt.Println("  ✅ RPATH set to $ORIGIN/lib")
		}
	} else {
		fmt.Println("  ⚠️  patchelf not found - ensure binary was compiled with RPATH=$ORIGIN/lib")
	}

	// Copy all libraries from AppDir/usr/lib to dist/lib
	libEntries, err := os.ReadDir(usrLibDir)
	if err != nil {
		return "", fmt.Errorf("reading lib directory: %w", err)
	}

	for _, entry := range libEntries {
		if entry.IsDir() {
			continue
		}
		srcLib := filepath.Join(usrLibDir, entry.Name())
		dstLib := filepath.Join(distLibDir, entry.Name())
		if err := copyFile(srcLib, dstLib); err != nil {
			return "", fmt.Errorf("copying library %s: %w", entry.Name(), err)
		}
	}
	fmt.Printf("  Copied %d libraries to lib/\n", len(libEntries))

	// Copy assets to dist root (excluding YAML level files)
	if _, err := os.Stat(config.AssetsDir); err == nil {
		assetsTarget := filepath.Join(distDir, "assets")
		if err := copyDirExcludingLevels(config.AssetsDir, assetsTarget); err != nil {
			return "", fmt.Errorf("copying assets to dist: %w", err)
		}
		fmt.Printf("  Copied assets to distribution directory (excluding YAML level files)\n")
	}

	// Copy protobuf level files from AppDir to dist (they were already written there)
	if config.LevelIterator != nil {
		appDirAssets := filepath.Join(usrShareDir, "assets", "levels")
		distAssets := filepath.Join(distDir, "assets", "levels")

		// Check if AppDir levels directory exists
		if _, err := os.Stat(appDirAssets); err == nil {
			// Copy the levels directory from AppDir to dist
			if err := copyDir(appDirAssets, distAssets); err != nil {
				return "", fmt.Errorf("copying level files to dist: %w", err)
			}
			fmt.Printf("  Copied protobuf level files to distribution directory\n")
		}
	}

	// Create zip archive in the build directory
	zipName := distDirName + ".zip"
	zipPath := filepath.Join(config.OutputDir, zipName)

	// Ensure output directory exists
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	fmt.Println("Creating zip archive...")
	if err := createZipArchive(distDir, zipPath, distDirName); err != nil {
		return "", fmt.Errorf("creating zip archive: %w", err)
	}

	fmt.Printf("\n✅ Linux package created: %s\n", zipPath)
	return zipPath, nil
}
