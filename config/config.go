package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`
	Database struct {
		FilePath        string        `mapstructure:"filepath"` // Path to the SQLite database file
		MaxOpenConns    int           `mapstructure:"max_open_conns"`
		MaxIdleConns    int           `mapstructure:"max_idle_conns"`
		ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	} `mapstructure:"database"`
}

type Database struct {
	FilePath        string        `mapstructure:"filepath"` // Path to the SQLite database file
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	viper.SetDefault("serverPort", "8080")

	viper.SetDefault("server.port", "8080")
	viper.SetDefault("database.filepath", "./data/transactions.db")
	viper.SetDefault("database.max_open_conns", 1)
	viper.SetDefault("database.max_idle_conns", 1)
	viper.SetDefault("database.conn_max_lifetime", "0s")

	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, fmt.Errorf("error reading config file: %w", err)
		}
		fmt.Println("Config file not found, using defaults/env vars.")
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return Config{}, fmt.Errorf("unable to decode into struct: %w", err)
	}

	return config, nil
}
