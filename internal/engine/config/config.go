// Package config provides configuration management for the application.
// config is built on top of the third-party module cleanenv
package config

import (
	"time"
)

type CompositorContract interface {
	LoadEnv() error
	LoadConf(path string) error
}

type Compositor struct {
	CMDLine *CMDLine
	Conf    *Conf
	Env     *Env
}

type Conf struct {
	Mode            string     `mapstructure:"mode"`
	ComDir          string     `mapstructure:"com_dir"`
	HTTPServer      HTTPServer `mapstructure:"http_server"`
	TLS             TLS        `mapstructure:"tls"`
	Updates         Updates    `mapstructure:"updates"`
	Log             Log        `mapstructure:"log"`
	DisableWarnings []string   `mapstructure:"disable_warnings"`
}

type HTTPServer struct {
	Address        string         `mapstructure:"address"`
	Port           string         `mapstructure:"port"`
	Timeout        time.Duration  `mapstructure:"timeout"`
	IdleTimeout    time.Duration  `mapstructure:"idle_timeout"`
}

type TLS struct {
	TlsEnabled bool   `mapstructure:"enabled"`
	CertFile   string `mapstructure:"cert_file"`
	KeyFile    string `mapstructure:"key_file"`
}

type Updates struct {
	UpdatesEnabled bool          `mapstructure:"enabled"`
	CheckInterval  time.Duration `mapstructure:"check_interval"`
	RepositoryURL  string        `mapstructure:"repository_url"`
	WantedVersion  string        `mapstructure:"wanted_version"`
}

type Log struct {
	Level   string `mapstructure:"level"`
	OutPath string `mapstructure:"out_path"`
}

// ConfigEnv structure for environment variables
type Env struct {
	ConfigPath     string `mapstructure:"config_path"`
	NodePath       string `mapstructure:"node_path"`
	ParentStagePID int    `mapstructure:"parent_pid"`
}

type CMDLine struct {
	Run  Run
	Node Root
}

type Root struct {
	Debug bool `persistent:"true" full:"debug" short:"d" def:"false" desc:"Set debug mode"`
}

type Run struct {
	ConfigPath string `persistent:"true" full:"config" short:"c" def:"./config.yaml" desc:"Path to configuration file"`
	Test       []int  `persistent:"true" full:"test" short:"t" def:"" desc:"js test"`
}
