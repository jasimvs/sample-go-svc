package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	ServerPort string `mapstructure:"serverPort"` // Port for the API server
	// Add other configurations here (e.g., Database DSN, external API keys)
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

	err = viper.ReadInConfig() // Find and read the config file
	if err != nil {
		// Handle errors reading the config file. If it's a "file not found" error,
		// it's okay if we have defaults or env vars set. Log other errors.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; ignore error if we have defaults/env vars
		fmt.Println("Config file not found, using defaults/env vars.")
	}

	err = viper.Unmarshal(&config) // Unmarshal config into struct
	if err != nil {
		return Config{}, fmt.Errorf("unable to decode into struct: %w", err)
	}

	return config, nil
}
