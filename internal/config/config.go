package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const defaultAPIURL = "http://localhost:8000"

type Config struct {
	APIURL string `json:"api_url"`
}

type Credentials struct {
	Token string `json:"token"`
}

func DefaultAPIURL() string { return defaultAPIURL }

func Dir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "ubinote"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "ubinote"), nil
}

func Load() (Config, error) {
	cfg := Config{APIURL: defaultAPIURL}
	dir, err := Dir()
	if err != nil {
		return cfg, err
	}
	b, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if v := os.Getenv("UBINOTE_API_URL"); v != "" {
				cfg.APIURL = v
			}
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.APIURL == "" {
		cfg.APIURL = defaultAPIURL
	}
	if v := os.Getenv("UBINOTE_API_URL"); v != "" {
		cfg.APIURL = v
	}
	return cfg, nil
}

func Save(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), b, 0o644)
}

func LoadCredentials() (Credentials, error) {
	var c Credentials
	dir, err := Dir()
	if err != nil {
		return c, err
	}
	b, err := os.ReadFile(filepath.Join(dir, "credentials.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c, nil
		}
		return c, err
	}
	return c, json.Unmarshal(b, &c)
}

func SaveCredentials(c Credentials) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "credentials.json")
	return os.WriteFile(path, b, 0o600)
}

func ClearCredentials() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(dir, "credentials.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
