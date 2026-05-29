//ff:what %APPDATA%/vpm/config.json 원자적 로드/저장
//ff:why 앱 크래시 시 설정 파일 부분 기록 방지 (write-to-temp then rename)
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configVersion = 1

// Dir returns the VPM config directory path.
func Dir() (string, error) {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return "", fmt.Errorf("APPDATA env not set")
	}
	return filepath.Join(appdata, "vpm"), nil
}

// Load reads and JSON-decodes the config file.
// Returns default config if the file does not exist.
func Load() (*Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaultConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// Save atomically writes cfg to the config file.
func Save(cfg *Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	path := filepath.Join(dir, "config.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write tmp config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

func defaultConfig() *Config {
	return &Config{
		Version:  configVersion,
		Settings: Settings{AutostartVPM: false, CloseWarningSeen: false},
		Projects: []Project{},
	}
}
