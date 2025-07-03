package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type ConfigConf struct {
	Mode       string `yaml:"mode" env-default:"dev"`
	ComDir     string `yaml:"com_dir" env-default:"./com/"`
	HTTPServer `yaml:"http_server"`
	TLS        `yaml:"tls"`
	Internal   `yaml:"internal"`
	Updates    `yaml:"updates"`
}

type Updates struct {
	UpdatesEnabled   bool          `yaml:"enabled" env-default:"false"`
	AllowAutoUpdates bool          `yaml:"allow_auto_updates" env-default:"false"`
	AllowUpdates     bool          `yaml:"allow_updates" env-default:"false"`
	AllowDowngrades  bool          `yaml:"allow_downgrades" env-default:"false"`
	CheckInterval    time.Duration `yaml:"check_interval" env-default:"2h"`
	RepositoryURL    string        `yaml:"repository_url" env-default:""`
	WantedVersion    string        `yaml:"wanted_version" env-default:"latest-stable"`
}

type Internal struct {
	MetaDir string `yaml:"meta_dir" env-default:"./.meta/"`
}

type TLS struct {
	TlsEnabled bool   `yaml:"enabled" env-default:"false"`
	CertFile   string `yaml:"cert_file" env-default:"./cert/server.crt"`
	KeyFile    string `yaml:"key_file" env-default:"./cert/server.key"`
}

type HTTPServer struct {
	Address        string        `yaml:"address" env-default:"0.0.0.0:8080"`
	Timeout        time.Duration `yaml:"timeout" env-default:"5s"`
	IdleTimeout    time.Duration `yaml:"idle_timeout" env-default:"60s"`
	HTTPServer_Api `yaml:"api"`
}

type HTTPServer_Api struct {
	LatestVer string   `yaml:"latest-version" env-required:"true"`
	Layers    []string `yaml:"layers"`
}

type ConfigEnv struct {
	ConfigPath string `env:"CONFIG_PATH" env-default:"./cfg/config.yaml"`
}

func MustLoadConfig() *ConfigConf {
	log.SetOutput(os.Stderr)
	var configEnv ConfigEnv
	if err := cleanenv.ReadEnv(&configEnv); err != nil {
		log.Fatalf("Failed to read environment variables: %v", err)
		os.Exit(1)
	}
	if _, err := os.Stat(configEnv.ConfigPath); os.IsNotExist(err) {
		log.Fatalf("Config file does not exist: %s", configEnv.ConfigPath)
		os.Exit(2)
	}
	var config ConfigConf
	if err := cleanenv.ReadConfig(configEnv.ConfigPath, &config); err != nil {
		log.Fatalf("Failed to read config file: %v", err)
		os.Exit(3)
	}
	log.Printf("Configuration loaded successfully from %s", configEnv.ConfigPath)
	return &config
}
