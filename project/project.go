package project

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const configFileName = "venture.yaml"

// Config represents the project configuration from venture.yaml.
type Config struct {
	Name       string `yaml:"name"`
	BinaryName string `yaml:"binary_name"`
	SteamAppID string `yaml:"steam_app_id,omitempty"`
}

// FindProjectRoot walks up from the current working directory looking for venture.yaml.
// Returns the directory containing venture.yaml, or an error if not found.
func FindProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}

	dir := cwd
	for {
		// Check if venture.yaml exists in this directory
		configPath := filepath.Join(dir, configFileName)
		if _, err := os.Stat(configPath); err == nil {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding venture.yaml
			return "", fmt.Errorf("venture.yaml not found in any parent directory of %s", cwd)
		}
		dir = parent
	}
}

// LoadConfig loads and parses the venture.yaml file from the given project root.
func LoadConfig(projectRoot string) (*Config, error) {
	configPath := filepath.Join(projectRoot, configFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", configFileName, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", configFileName, err)
	}

	// Validate required fields
	if config.Name == "" {
		return nil, fmt.Errorf("'name' field is required in %s", configFileName)
	}
	if config.BinaryName == "" {
		return nil, fmt.Errorf("'binary_name' field is required in %s", configFileName)
	}

	return &config, nil
}
