package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.APIID != 0 {
		t.Error("default api_id should be 0")
	}
	if cfg.Proxy.Type != "socks5" {
		t.Error("default proxy should be socks5")
	}
	if cfg.Limits.MaxConcurrent != 5 {
		t.Errorf("max_concurrent = %d, want 5", cfg.Limits.MaxConcurrent)
	}
}

func TestSaveLoad(t *testing.T) {
	cfg := DefaultConfig()
	cfg.APIID = 12345
	cfg.APIHash = "testhash123"
	// Use temp config file
	orig := ConfigFile
	ConfigFile = os.TempDir() + "/tgtool_test_config.json"
	defer os.Remove(ConfigFile)
	defer func() { ConfigFile = orig }()

	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	loaded := Load()
	if loaded.APIID != 12345 {
		t.Errorf("api_id = %d, want 12345", loaded.APIID)
	}
	if loaded.APIHash != "testhash123" {
		t.Errorf("api_hash = %s, want testhash123", loaded.APIHash)
	}
}

func TestDirsExist(t *testing.T) {
	for _, d := range []string{SessionsDir, DataDir, LogsDir, MediaDir} {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			t.Errorf("dir missing: %s", d)
		}
	}
}
