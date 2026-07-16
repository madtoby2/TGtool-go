package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ProxyConfig struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LimitsConfig struct {
	MaxConcurrent   int     `json:"max_concurrent"`
	JoinIntervalMin float64 `json:"join_interval_min"`
	JoinIntervalMax float64 `json:"join_interval_max"`
	SendIntervalMin float64 `json:"send_interval_min"`
	SendIntervalMax float64 `json:"send_interval_max"`
	DmPerAccount    int     `json:"dm_per_account"`
	InviteInterval  float64 `json:"invite_interval"`
}

type Config struct {
	APIID   int         `json:"api_id"`
	APIHash string      `json:"api_hash"`
	Proxy   ProxyConfig `json:"proxy"`
	Limits  LimitsConfig `json:"limits"`
}

func DefaultConfig() Config {
	return Config{
		Proxy: ProxyConfig{
			Type: "socks5",
			Host: "127.0.0.1",
			Port: 1080,
		},
		Limits: LimitsConfig{
			MaxConcurrent:   5,
			JoinIntervalMin: 30,
			JoinIntervalMax: 60,
			SendIntervalMin: 10,
			SendIntervalMax: 30,
			DmPerAccount:    30,
			InviteInterval:  30,
		},
	}
}

var (
	RootDir     string
	SessionsDir string
	DataDir     string
	LogsDir     string
	MediaDir    string
	ConfigFile  string
)

func init() {
	exe, _ := os.Executable()
	RootDir = filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(RootDir, "config.json")); os.IsNotExist(err) {
		RootDir, _ = os.Getwd()
	}
	SessionsDir = filepath.Join(RootDir, "sessions")
	DataDir = filepath.Join(RootDir, "data")
	LogsDir = filepath.Join(RootDir, "logs")
	MediaDir = filepath.Join(RootDir, "media")
	ConfigFile = filepath.Join(RootDir, "config.json")
	for _, d := range []string{SessionsDir, DataDir, LogsDir, MediaDir} {
		os.MkdirAll(d, 0755)
	}
}

func Load() Config {
	cfg := DefaultConfig()
	if data, err := os.ReadFile(ConfigFile); err == nil {
		json.Unmarshal(data, &cfg)
	}
	return cfg
}

func Save(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFile, data, 0644)
}
