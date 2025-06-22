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
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"0.0.0.0:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"5s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

type ConfigEnv struct {
	ConfigPath string `env:"CONFIG_PATH" env-default:"./cfg/config.yaml"`
}

func MustLoadConfig() *ConfigConf {
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
