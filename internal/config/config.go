package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	HostID                 string `json:"host_id"`
	HostToken              string `json:"host_token"`
	ServerURL              string `json:"server_url"`
	AllowInsecureLocalhost bool   `json:"allow_insecure_localhost,omitempty"`
}

func Load(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := json.Unmarshal(content, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	content, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(path, content, 0o600)
}
