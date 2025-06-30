package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected Config
	}{
		{
			name:   "empty config should set all defaults",
			config: Config{},
			expected: Config{
				Port:         "8080",
				KUBECONFIG:   "~/.kube/config",
				LoggingLevel: "info",
			},
		},
		{
			name: "partial config should set missing defaults",
			config: Config{
				Port: "9090",
			},
			expected: Config{
				Port:         "9090",
				KUBECONFIG:   "~/.kube/config",
				LoggingLevel: "info",
			},
		},
		{
			name: "full config should not change",
			config: Config{
				Port:         "9090",
				KUBECONFIG:   "/custom/kube/config",
				LoggingLevel: "debug",
			},
			expected: Config{
				Port:         "9090",
				KUBECONFIG:   "/custom/kube/config",
				LoggingLevel: "debug",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.setDefaults()
			if tt.config != tt.expected {
				t.Errorf("setDefaults() = %v, want %v", tt.config, tt.expected)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	// Test that GetConfigPath returns a valid path
	path := GetConfigPath()
	if path == "" {
		t.Error("GetConfigPath() returned empty path")
	}

	// Test that the path is either absolute or relative
	if !filepath.IsAbs(path) && !filepath.IsLocal(path) {
		t.Errorf("GetConfigPath() returned invalid path: %s", path)
	}
}

func TestConfig_PrintConfig(t *testing.T) {
	config := Config{
		Port:         "8080",
		KUBECONFIG:   "~/.kube/config",
		LoggingLevel: "info",
	}

	// This test mainly ensures PrintConfig doesn't panic
	// In a real scenario, you might want to capture stdout and verify the output
	config.PrintConfig()
}

func TestLoadConfig_WithEnvFile(t *testing.T) {
	// Reset Viper to clear any cached values
	viper.Reset()

	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Create a test .env file
	envContent := `PORT=9090
LOGGING_LEVEL=debug
KUBECONFIG=/test/kube/config`

	envFile := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Load config from the test directory
	config, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify the loaded values
	expected := Config{
		Port:         "9090",
		KUBECONFIG:   "/test/kube/config",
		LoggingLevel: "debug",
	}

	if config != expected {
		t.Errorf("LoadConfig() = %v, want %v", config, expected)
	}
}

func TestLoadConfig_WithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	if err := os.Setenv("PORT", "7070"); err != nil {
		t.Fatalf("Failed to set PORT env var: %v", err)
	}
	if err := os.Setenv("LOGGING_LEVEL", "warn"); err != nil {
		t.Fatalf("Failed to set LOGGING_LEVEL env var: %v", err)
	}
	if err := os.Setenv("KUBECONFIG", "/env/kube/config"); err != nil {
		t.Fatalf("Failed to set KUBECONFIG env var: %v", err)
	}

	// Clean up after test
	defer func() {
		if err := os.Unsetenv("PORT"); err != nil {
			t.Errorf("Failed to unset PORT env var: %v", err)
		}
		if err := os.Unsetenv("LOGGING_LEVEL"); err != nil {
			t.Errorf("Failed to unset LOGGING_LEVEL env var: %v", err)
		}
		if err := os.Unsetenv("KUBECONFIG"); err != nil {
			t.Errorf("Failed to unset KUBECONFIG env var: %v", err)
		}
	}()

	// Load config (should use environment variables)
	config, err := LoadConfig("nonexistent/path")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify the loaded values
	expected := Config{
		Port:         "7070",
		KUBECONFIG:   "/env/kube/config",
		LoggingLevel: "warn",
	}

	if config != expected {
		t.Errorf("LoadConfig() = %v, want %v", config, expected)
	}
}

func TestLoadConfig_WithDefaults(t *testing.T) {
	// Reset Viper to clear any cached values
	viper.Reset()

	// Clear environment variables to test defaults
	originalPort := os.Getenv("PORT")
	originalLoggingLevel := os.Getenv("LOGGING_LEVEL")
	originalKubeconfig := os.Getenv("KUBECONFIG")

	if err := os.Unsetenv("PORT"); err != nil {
		t.Fatalf("Failed to unset PORT env var: %v", err)
	}
	if err := os.Unsetenv("LOGGING_LEVEL"); err != nil {
		t.Fatalf("Failed to unset LOGGING_LEVEL env var: %v", err)
	}
	if err := os.Unsetenv("KUBECONFIG"); err != nil {
		t.Fatalf("Failed to unset KUBECONFIG env var: %v", err)
	}

	// Restore original values after test
	defer func() {
		if originalPort != "" {
			if err := os.Setenv("PORT", originalPort); err != nil {
				t.Errorf("Failed to restore PORT env var: %v", err)
			}
		}
		if originalLoggingLevel != "" {
			if err := os.Setenv("LOGGING_LEVEL", originalLoggingLevel); err != nil {
				t.Errorf("Failed to restore LOGGING_LEVEL env var: %v", err)
			}
		}
		if originalKubeconfig != "" {
			if err := os.Setenv("KUBECONFIG", originalKubeconfig); err != nil {
				t.Errorf("Failed to restore KUBECONFIG env var: %v", err)
			}
		}
	}()

	// Load config from nonexistent path (should use defaults)
	config, err := LoadConfig("nonexistent/path")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify default values are set
	if config.Port != "8080" {
		t.Errorf("Expected default PORT=8080, got %s", config.Port)
	}
	if config.KUBECONFIG != "~/.kube/config" {
		t.Errorf("Expected default KUBECONFIG=~/.kube/config, got %s", config.KUBECONFIG)
	}
	if config.LoggingLevel != "info" {
		t.Errorf("Expected default LOGGING_LEVEL=info, got %s", config.LoggingLevel)
	}
}

func TestLoadConfig_EmptyEnvFile(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Create an empty .env file
	envFile := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Load config should work with empty file (should use defaults)
	config, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Should use default values
	if config.Port != "8080" {
		t.Errorf("Expected default PORT=8080, got %s", config.Port)
	}
	if config.KUBECONFIG != "~/.kube/config" {
		t.Errorf("Expected default KUBECONFIG=~/.kube/config, got %s", config.KUBECONFIG)
	}
	if config.LoggingLevel != "info" {
		t.Errorf("Expected default LOGGING_LEVEL=info, got %s", config.LoggingLevel)
	}
}
