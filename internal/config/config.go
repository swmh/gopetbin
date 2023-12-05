package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

const configSample = "./config-sample.yml"

//go:generate go run github.com/swmh/gopetbin/pkg/vipergen --path ../../config/ --name ".env.sample" -env
//go:generate go run github.com/swmh/gopetbin/pkg/vipergen --path ../../config/ --name "config-sample.yml" -yml
type Config struct {
	App struct {
		Addr              string `mapstructure:"addr"`
		IDLength          int    `mapstructure:"id_length"`
		MaxSize           int64  `mapstructure:"max_size"` /* max paste size in bytes */
		ReadTimeout       int    `mapstructure:"timeout_read"`
		WriteTimeout      int    `mapstructure:"timeout_write"`
		LogLevel          string `mapstructure:"log_level"` /* debug, info, warn, error */
		PublicPath        string `mapstructure:"public_path"`
		MaxFileMemory     int64  `mapstructure:"max_file_memory"`    /* maximum bytes of file data stored in memory */
		DefaultExpiration int    `mapstructure:"default_expiration"` /* hours */
	} `mapstructure:"app"`

	DB struct {
		Addr string `mapstructure:"addr"`
		User string `mapstructure:"user"`
		Pass string `mapstructure:"pass"`
		Name string `mapstructure:"name"`
	} `mapstructure:"db"`

	Storage struct {
		Addr string `mapstructure:"addr"`
		User string `mapstructure:"user"`
		Pass string `mapstructure:"pass"`
		Name string `mapstructure:"name"`
	} `mapstructure:"storage"`

	Cache struct {
		Addr string `mapstructure:"addr"`
		User string `mapstructure:"user"`
		Pass string `mapstructure:"pass"`
		DB   int    `mapstructure:"db"`
	} `mapstructure:"cache"`

	FileCache struct {
		Addr string `mapstructure:"addr"`
		User string `mapstructure:"user"`
		Pass string `mapstructure:"pass"`
		DB   int    `mapstructure:"db"`
	} `mapstructure:"file_cache"`

	Locker struct {
		Addr string `mapstructure:"addr"`
		User string `mapstructure:"user"`
		Pass string `mapstructure:"pass"`
		DB   int    `mapstructure:"db"`
	} `mapstructure:"locker"`
}

func New(path string) (*Config, error) {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.SetConfigType("yaml")

	viper.SetDefault("app.log_level", "info")
	viper.SetDefault("app.addr", ":80")
	viper.SetDefault("app.max_size", "10485760")
	viper.SetDefault("app.id_length", "10")

	viper.SetConfigFile(configSample)
	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot read sample config file: %w", err)
	}

	if path != "" {
		viper.SetConfigFile(path)
		err = viper.MergeInConfig()
		if err != nil {
			return nil, fmt.Errorf("cannot read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("cannot unmarshal config: %w", err)
	}

	return &config, nil
}
