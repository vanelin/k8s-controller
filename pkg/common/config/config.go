package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Port                    string `mapstructure:"PORT"`
	KUBECONFIG              string `mapstructure:"KUBECONFIG"`
	LoggingLevel            string `mapstructure:"LOGGING_LEVEL"`
	Namespace               string `mapstructure:"NAMESPACE"`
	InCluster               bool   `mapstructure:"IN_CLUSTER"`
	MetricPort              string `mapstructure:"METRIC_PORT"`
	EnableLeaderElection    bool   `mapstructure:"ENABLE_LEADER_ELECTION"`
	LeaderElectionNamespace string `mapstructure:"LEADER_ELECTION_NAMESPACE"`
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
	if err := viper.BindEnv("NAMESPACE"); err != nil {
		return config, fmt.Errorf("failed to bind NAMESPACE env var: %w", err)
	}
	if err := viper.BindEnv("IN_CLUSTER"); err != nil {
		return config, fmt.Errorf("failed to bind IN_CLUSTER env var: %w", err)
	}
	if err := viper.BindEnv("METRIC_PORT"); err != nil {
		return config, fmt.Errorf("failed to bind METRIC_PORT env var: %w", err)
	}
	if err := viper.BindEnv("ENABLE_LEADER_ELECTION"); err != nil {
		return config, fmt.Errorf("failed to bind ENABLE_LEADER_ELECTION env var: %w", err)
	}
	if err := viper.BindEnv("LEADER_ELECTION_NAMESPACE"); err != nil {
		return config, fmt.Errorf("failed to bind LEADER_ELECTION_NAMESPACE env var: %w", err)
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
	if c.Namespace == "" {
		c.Namespace = "default"
	}
	if c.MetricPort == "" {
		c.MetricPort = "8081"
	}
	// Only set EnableLeaderElection default if it wasn't set via viper
	// This allows the test to work correctly when viper is not used
	if !viper.IsSet("ENABLE_LEADER_ELECTION") {
		c.EnableLeaderElection = true
	}
	if c.LeaderElectionNamespace == "" {
		c.LeaderElectionNamespace = "default"
	}
	// InCluster defaults to false, no need to set it
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
	fmt.Printf("  METRIC_PORT: %s\n", c.MetricPort)
	fmt.Printf("  LOGGING_LEVEL: %s\n", c.LoggingLevel)
	if c.KUBECONFIG != "" {
		fmt.Printf("  KUBECONFIG: %s\n", c.KUBECONFIG)
	} else {
		fmt.Printf("  KUBECONFIG: [NOT SET]\n")
	}
	fmt.Printf("  NAMESPACE: %s\n", c.Namespace)
	fmt.Printf("  IN_CLUSTER: %t\n", c.InCluster)
	fmt.Printf("  ENABLE_LEADER_ELECTION: %t\n", c.EnableLeaderElection)
	fmt.Printf("  LEADER_ELECTION_NAMESPACE: %s\n", c.LeaderElectionNamespace)
}
