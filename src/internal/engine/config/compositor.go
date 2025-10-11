package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
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
	v.SetDefault("node.name", "noname")
	v.SetDefault("node.mode", "dev")
	v.SetDefault("node.show_config", "false")
	v.SetDefault("node.com_dir", "./com/")
	v.SetDefault("http_server.address", "0.0.0.0")
	v.SetDefault("http_server.port", "8080")
	v.SetDefault("http_server.session_ttl", "30m")
	v.SetDefault("http_server.timeout", "5s")
	v.SetDefault("http_server.idle_timeout", "60s")
	v.SetDefault("tls.enabled", false)
	v.SetDefault("tls.cert_file", "./cert/server.crt")
	v.SetDefault("tls.key_file", "./cert/server.key")
	v.SetDefault("updates.enabled", false)
	v.SetDefault("updates.check_interval", "2h")
	v.SetDefault("updates.wanted_version", "latest-stable")
	v.SetDefault("log.json_format", "false")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.output", "%2%")
	v.SetDefault("disable_warnings", []string{})

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config: %w", err)
	}

	var cfg Conf
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}

	c.Conf = &Conf{}
	c.Conf = &cfg
	return nil
}

func (c *Compositor) LoadCMDLine(root *cobra.Command) {
	cmdLine := &CMDLine{}
	c.CMDLine = cmdLine

	t := reflect.TypeOf(cmdLine).Elem()
	v := reflect.ValueOf(cmdLine).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)
		ptr := fieldVal.Addr().Interface()
		use := strings.ToLower(field.Name)

		var cmd *cobra.Command
		for _, sub := range root.Commands() {

			if sub.Use == use {
				cmd = sub
				break
			}
		}

		if use == root.Use {
			cmd = root
		}

		if cmd == nil {
			continue
		}

		Unmarshal(cmd, ptr)
	}
}

func Unmarshal(cmd *cobra.Command, target any) {
	t := reflect.TypeOf(target).Elem()
	v := reflect.ValueOf(target).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		valPtr := v.Field(i).Addr().Interface()

		full := field.Tag.Get("full")
		short := field.Tag.Get("short")
		def := field.Tag.Get("def")
		desc := field.Tag.Get("desc")
		isPersistent := field.Tag.Get("persistent") == "true"

		flagSet := cmd.Flags()
		if isPersistent {
			flagSet = cmd.PersistentFlags()
		}

		switch field.Type.Kind() {
		case reflect.String:
			flagSet.StringVarP(valPtr.(*string), full, short, def, desc)

		case reflect.Bool:
			defVal, err := strconv.ParseBool(def)
			if err != nil && def != "" {
				fmt.Printf("warning: cannot parse default bool: %q\n", def)
			}
			flagSet.BoolVarP(valPtr.(*bool), full, short, defVal, desc)

		case reflect.Int:
			defVal, err := strconv.Atoi(def)
			if err != nil && def != "" {
				fmt.Printf("warning: cannot parse default int: %q\n", def)
			}
			flagSet.IntVarP(valPtr.(*int), full, short, defVal, desc)

		case reflect.Slice:
			elemKind := field.Type.Elem().Kind()
			switch elemKind {
			case reflect.String:
				defVals := []string{}
				if def != "" {
					defVals = strings.Split(def, ",")
				}
				flagSet.StringSliceVarP(valPtr.(*[]string), full, short, defVals, desc)

			case reflect.Int:
				var intVals []int
				if def != "" {
					for _, s := range strings.Split(def, ",") {
						s = strings.TrimSpace(s)
						if s == "" {
							continue
						}
						n, err := strconv.Atoi(s)
						if err != nil {
							fmt.Printf("warning: cannot parse int in slice: %q\n", s)
							continue
						}
						intVals = append(intVals, n)
					}
				}
				flagSet.IntSliceVarP(valPtr.(*[]int), full, short, intVals, desc)

			default:
				fmt.Printf("unsupported slice element type: %s\n", elemKind)
			}

		default:
			fmt.Printf("unsupported field type: %s\n", field.Type.Kind())
		}
	}
}
