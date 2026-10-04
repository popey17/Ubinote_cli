package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"ubinote-cli/internal/config"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("UBINOTE_API_URL", "")

	cfg := config.Config{APIURL: "http://example.com:9000"}
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.APIURL != "http://example.com:9000" {
		t.Fatalf("got %q", got.APIURL)
	}
}

func TestEnvOverridesAPIURL(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("UBINOTE_API_URL", "http://override:1")
	_ = config.Save(config.Config{APIURL: "http://file:2"})
	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.APIURL != "http://override:1" {
		t.Fatalf("got %q", got.APIURL)
	}
}

func TestCredentialsMode0600(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.SaveCredentials(config.Credentials{Token: "abc"}); err != nil {
		t.Fatal(err)
	}
	dir, _ := config.Dir()
	info, err := os.Stat(filepath.Join(dir, "credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
}
