package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Port         string `mapstructure:"PORT"`
	KUBECONFIG   string `mapstructure:"KUBECONFIG"`
	LoggingLevel string `mapstructure:"LOGGING_LEVEL"`
}

// LoadConfig reads configuration from file or environment variables
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	// Bind environment variables to config fields
	if err := viper.BindEnv("PORT"); err != nil {
		return config, fmt.Errorf("failed to bind PORT env var: %w", err)
	}
	if err := viper.BindEnv("KUBECONFIG"); err != nil {
		return config, fmt.Errorf("failed to bind KUBECONFIG env var: %w", err)
	}
	if err := viper.BindEnv("LOGGING_LEVEL"); err != nil {
		return config, fmt.Errorf("failed to bind LOGGING_LEVEL env var: %w", err)
	}

	// Enable automatic environment variable reading
	viper.AutomaticEnv()

	// Read .env file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, continue with environment variables only
	}

	// Unmarshal config into struct
	if err := viper.Unmarshal(&config); err != nil {
		return config, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set default values for empty fields
	config.setDefaults()

	return config, nil
}

// setDefaults sets default values for empty configuration fields
func (c *Config) setDefaults() {
	if c.Port == "" {
		c.Port = "8080"
	}
	if c.KUBECONFIG == "" {
		c.KUBECONFIG = "~/.kube/config"
	}
	if c.LoggingLevel == "" {
		c.LoggingLevel = "info"
	}
}

// GetConfigPath returns the path to the config directory
func GetConfigPath() string {
	// Try to get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		// Fallback to relative path
		return "pkg/common/envs"
	}

	// Look for config in the project structure
	configPath := filepath.Join(wd, "pkg", "common", "envs")
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}

	// If not found, return relative path
	return "pkg/common/envs"
}

// PrintConfig prints the current configuration (without sensitive data)
func (c *Config) PrintConfig() {
	fmt.Printf("Configuration:\n")
	fmt.Printf("  PORT: %s\n", c.Port)
	fmt.Printf("  LOGGING_LEVEL: %s\n", c.LoggingLevel)
	if c.KUBECONFIG != "" {
		fmt.Printf("  KUBECONFIG: %s\n", c.KUBECONFIG)
	} else {
		fmt.Printf("  KUBECONFIG: [NOT SET]\n")
	}
}
