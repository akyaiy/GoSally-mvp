package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func NewCompositor() *Compositor {
	return &Compositor{}
}

func (c *Compositor) LoadEnv() error {
	v := viper.New()

	// defaults
	v.SetDefault("config_path", "./cfg/config.yaml")
	v.SetDefault("node_path", "./")
	v.SetDefault("parent_pid", -1)

	// GS_*
	v.SetEnvPrefix("GS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var env Env
	if err := v.Unmarshal(&env); err != nil {
		return fmt.Errorf("error unmarshaling env: %w", err)
	}

	c.Env = &env
	return nil
}

func (c *Compositor) LoadConf(path string) error {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// defaults
	v.SetDefault("mode", "dev")
	v.SetDefault("com_dir", "./com/")
	v.SetDefault("http_server.address", "0.0.0.0:8080")
	v.SetDefault("http_server.timeout", "5s")
	v.SetDefault("http_server.idle_timeout", "60s")
	v.SetDefault("tls.enabled", false)
	v.SetDefault("tls.cert_file", "./cert/server.crt")
	v.SetDefault("tls.key_file", "./cert/server.key")
	v.SetDefault("updates.enabled", false)
	v.SetDefault("updates.check_interval", "2h")
	v.SetDefault("updates.wanted_version", "latest-stable")

	// поддержка ENV-переопределений
	v.SetEnvPrefix("GOSALLY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// читаем YAML
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config: %w", err)
	}

	var cfg Conf
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}

	c.Conf = &cfg
	return nil
}
