package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Config holds the application configuration
type Config struct {
	GeniusAccessToken string `json:"genius_access_token"`
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	// Check XDG_CONFIG_HOME first
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		// Fall back to ~/.config
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", errors.Wrap(err, "could not determine home directory")
		}
		configHome = filepath.Join(homeDir, ".config")
	}

	// Create app config directory if it doesn't exist
	appConfigDir := filepath.Join(configHome, "lyrics")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", errors.Wrap(err, "create config directory")
	}

	return filepath.Join(appConfigDir, "config.json"), nil
}

// LoadConfig loads the configuration from the config file
func LoadConfig() (Config, error) {
	var config Config

	configPath, err := getConfigPath()
	if err != nil {
		return config, errors.Wrap(err, "get config path")
	}

	// Read and parse config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist yet, which is okay
			return config, nil
		}
		return config, errors.Wrap(err, "read config file")
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return config, errors.Wrap(err, "parse config file")
	}

	return config, nil
}
