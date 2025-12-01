package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type InstanceConfig struct {
	BackendURL     string `json:"backend_url"`
	InstanceKey    string `json:"instance_key"`
	InstanceSecret string `json:"instance_secret"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "watcher-client", "config.json"), nil
}

func Load() (*InstanceConfig, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &InstanceConfig{}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg InstanceConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(cfg *InstanceConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
