package packager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// packagePlatform creates an AppImage using linuxdeploy
func packagePlatform(config PackageConfig) (string, error) {
	fmt.Println("Creating Linux AppImage...")

	// Check if linuxdeploy is available
	linuxdeployPath, err := exec.LookPath("linuxdeploy")
	if err != nil {
		return "", fmt.Errorf("linuxdeploy not found. Download it from: https://github.com/linuxdeploy/linuxdeploy/releases")
	}

	// Create AppDir structure
	appDirName := config.BinaryName + ".AppDir"
	appDir := filepath.Join(config.OutputDir, appDirName)
	usrBinDir := filepath.Join(appDir, "usr", "bin")
	usrLibDir := filepath.Join(appDir, "usr", "lib")
	usrShareDir := filepath.Join(appDir, "usr", "share")

	// Remove old AppDir if it exists
	if err := os.RemoveAll(appDir); err != nil {
		return "", fmt.Errorf("removing old AppDir: %w", err)
	}

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

	// Copy assets
	if _, err := os.Stat(config.AssetsDir); err == nil {
		assetsTarget := filepath.Join(usrShareDir, "assets")
		if err := copyDir(config.AssetsDir, assetsTarget); err != nil {
			return "", fmt.Errorf("copying assets: %w", err)
		}
		fmt.Printf("  Copied assets\n")
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

	// Create .desktop file
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
	// In a real project, you'd have an actual icon
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

	// Run linuxdeploy to create AppImage
	fmt.Println("Running linuxdeploy to create AppImage and bundle libraries...")

	outputFileName := fmt.Sprintf("%s-%s.AppImage", config.BinaryName, config.Target)
	outputPath := filepath.Join(config.OutputDir, outputFileName)

	// Remove old AppImage if it exists
	os.Remove(outputPath)

	cmd := exec.Command(linuxdeployPath,
		"--appdir", appDir,
		"--output", "appimage",
		"--executable", targetBinary,
	)
	cmd.Dir = config.OutputDir
	cmd.Env = append(os.Environ(), "OUTPUT="+outputFileName)

	output, err := cmd.CombinedOutput()
	fmt.Printf("linuxdeploy output:\n%s\n", string(output))

	if err != nil {
		return "", fmt.Errorf("running linuxdeploy: %w\nOutput: %s", err, string(output))
	}

	// linuxdeploy creates the AppImage in the output directory
	// Find the generated AppImage
	matches, err := filepath.Glob(filepath.Join(config.OutputDir, "*.AppImage"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("AppImage not found after linuxdeploy execution")
	}

	appImagePath := matches[0]

	// Make sure it's executable
	if err := os.Chmod(appImagePath, 0755); err != nil {
		return "", fmt.Errorf("making AppImage executable: %w", err)
	}

	fmt.Printf("\n✅ Linux AppImage created: %s\n", appImagePath)
	return appImagePath, nil
}
