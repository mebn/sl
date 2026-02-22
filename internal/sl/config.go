package sl

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func loadConfig() (appConfig, error) {
	path, err := configPath()
	if err != nil {
		return appConfig{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return appConfig{}, err
	}

	var cfg appConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return appConfig{}, err
	}

	return cfg, nil
}

func SaveRoute(from, to string) error {
	return saveConfig(appConfig{From: from, To: to})
}

func saveConfig(cfg appConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	return nil
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "sl", "config.json"), nil
}
