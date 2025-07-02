package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/vanelin/k8s-controller.git/pkg/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				Namespace:    "default",
				InCluster:    false,
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
				Namespace:    "default",
				InCluster:    false,
			},
		},
		{
			name: "full config should not change",
			config: Config{
				Port:         "9090",
				KUBECONFIG:   "/custom/kube/config",
				LoggingLevel: "debug",
				Namespace:    "custom-namespace",
				InCluster:    true,
			},
			expected: Config{
				Port:         "9090",
				KUBECONFIG:   "/custom/kube/config",
				LoggingLevel: "debug",
				Namespace:    "custom-namespace",
				InCluster:    true,
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
		Namespace:    "test-namespace",
		InCluster:    false,
	}

	// This test mainly ensures PrintConfig doesn't panic
	// In a real scenario, you might want to capture stdout and verify the output
	config.PrintConfig()
}

func TestLoadConfig_WithEnvFile(t *testing.T) {
	// Reset Viper to clear any cached values
	viper.Reset()

	// Clear environment variables to ensure .env file values are used
	originalPort := os.Getenv("PORT")
	originalLoggingLevel := os.Getenv("LOGGING_LEVEL")
	originalKubeconfig := os.Getenv("KUBECONFIG")
	originalNamespace := os.Getenv("NAMESPACE")
	originalInCluster := os.Getenv("IN_CLUSTER")

	if err := os.Unsetenv("PORT"); err != nil {
		t.Fatalf("Failed to unset PORT env var: %v", err)
	}
	if err := os.Unsetenv("LOGGING_LEVEL"); err != nil {
		t.Fatalf("Failed to unset LOGGING_LEVEL env var: %v", err)
	}
	if err := os.Unsetenv("KUBECONFIG"); err != nil {
		t.Fatalf("Failed to unset KUBECONFIG env var: %v", err)
	}
	if err := os.Unsetenv("NAMESPACE"); err != nil {
		t.Fatalf("Failed to unset NAMESPACE env var: %v", err)
	}
	if err := os.Unsetenv("IN_CLUSTER"); err != nil {
		t.Fatalf("Failed to unset IN_CLUSTER env var: %v", err)
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
		if originalNamespace != "" {
			if err := os.Setenv("NAMESPACE", originalNamespace); err != nil {
				t.Errorf("Failed to restore NAMESPACE env var: %v", err)
			}
		}
		if originalInCluster != "" {
			if err := os.Setenv("IN_CLUSTER", originalInCluster); err != nil {
				t.Errorf("Failed to restore IN_CLUSTER env var: %v", err)
			}
		}
	}()

	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Create a test .env file
	envContent := `PORT=9090
LOGGING_LEVEL=debug
KUBECONFIG=/test/kube/config
NAMESPACE=test-namespace
IN_CLUSTER=true`

	envFile := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Load config from the test directory
	config, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify the loaded values from .env file (no environment variables to override)
	expected := Config{
		Port:         "9090",              // From .env file
		KUBECONFIG:   "/test/kube/config", // From .env file
		LoggingLevel: "debug",             // From .env file
		Namespace:    "test-namespace",    // From .env file
		InCluster:    true,                // From .env file
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
	if err := os.Setenv("NAMESPACE", "env-namespace"); err != nil {
		t.Fatalf("Failed to set NAMESPACE env var: %v", err)
	}
	if err := os.Setenv("IN_CLUSTER", "true"); err != nil {
		t.Fatalf("Failed to set IN_CLUSTER env var: %v", err)
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
		if err := os.Unsetenv("NAMESPACE"); err != nil {
			t.Errorf("Failed to unset NAMESPACE env var: %v", err)
		}
		if err := os.Unsetenv("IN_CLUSTER"); err != nil {
			t.Errorf("Failed to unset IN_CLUSTER env var: %v", err)
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
		Namespace:    "env-namespace",
		InCluster:    true,
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
	originalNamespace := os.Getenv("NAMESPACE")
	originalInCluster := os.Getenv("IN_CLUSTER")

	if err := os.Unsetenv("PORT"); err != nil {
		t.Fatalf("Failed to unset PORT env var: %v", err)
	}
	if err := os.Unsetenv("LOGGING_LEVEL"); err != nil {
		t.Fatalf("Failed to unset LOGGING_LEVEL env var: %v", err)
	}
	if err := os.Unsetenv("KUBECONFIG"); err != nil {
		t.Fatalf("Failed to unset KUBECONFIG env var: %v", err)
	}
	if err := os.Unsetenv("NAMESPACE"); err != nil {
		t.Fatalf("Failed to unset NAMESPACE env var: %v", err)
	}
	if err := os.Unsetenv("IN_CLUSTER"); err != nil {
		t.Fatalf("Failed to unset IN_CLUSTER env var: %v", err)
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
		if originalNamespace != "" {
			if err := os.Setenv("NAMESPACE", originalNamespace); err != nil {
				t.Errorf("Failed to restore NAMESPACE env var: %v", err)
			}
		}
		if originalInCluster != "" {
			if err := os.Setenv("IN_CLUSTER", originalInCluster); err != nil {
				t.Errorf("Failed to restore IN_CLUSTER env var: %v", err)
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
	if config.Namespace != "default" {
		t.Errorf("Expected default NAMESPACE=default, got %s", config.Namespace)
	}
	if config.InCluster != false {
		t.Errorf("Expected default IN_CLUSTER=false, got %t", config.InCluster)
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
	if config.Namespace != "default" {
		t.Errorf("Expected default NAMESPACE=default, got %s", config.Namespace)
	}
	if config.InCluster != false {
		t.Errorf("Expected default IN_CLUSTER=false, got %t", config.InCluster)
	}
}

// TestLoadConfig_EnvOverridesEnvFile explicitly tests that environment variables override .env file values
func TestLoadConfig_EnvOverridesEnvFile(t *testing.T) {
	// Reset Viper to clear any cached values
	viper.Reset()

	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Create a .env file with one set of values
	envContent := `PORT=9090
LOGGING_LEVEL=debug
KUBECONFIG=/test/kube/config
NAMESPACE=test-namespace
IN_CLUSTER=false`

	envFile := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Set environment variables with different values
	if err := os.Setenv("PORT", "8030"); err != nil {
		t.Fatalf("Failed to set PORT env var: %v", err)
	}
	if err := os.Setenv("LOGGING_LEVEL", "warn"); err != nil {
		t.Fatalf("Failed to set LOGGING_LEVEL env var: %v", err)
	}
	if err := os.Setenv("KUBECONFIG", "/env/kube/config"); err != nil {
		t.Fatalf("Failed to set KUBECONFIG env var: %v", err)
	}
	if err := os.Setenv("NAMESPACE", "env-namespace"); err != nil {
		t.Fatalf("Failed to set NAMESPACE env var: %v", err)
	}
	if err := os.Setenv("IN_CLUSTER", "true"); err != nil {
		t.Fatalf("Failed to set IN_CLUSTER env var: %v", err)
	}

	// Clean up environment variables after test
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
		if err := os.Unsetenv("NAMESPACE"); err != nil {
			t.Errorf("Failed to unset NAMESPACE env var: %v", err)
		}
		if err := os.Unsetenv("IN_CLUSTER"); err != nil {
			t.Errorf("Failed to unset IN_CLUSTER env var: %v", err)
		}
	}()

	// Load config from the test directory
	config, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify that environment variables override .env file values
	expected := Config{
		Port:         "8030",             // From PORT env var
		KUBECONFIG:   "/env/kube/config", // From KUBECONFIG env var
		LoggingLevel: "warn",             // From LOGGING_LEVEL env var
		Namespace:    "env-namespace",    // From NAMESPACE env var
		InCluster:    true,               // From IN_CLUSTER env var
	}

	if config != expected {
		t.Errorf("LoadConfig() = %v, want %v", config, expected)
	}
}

// TestLoadConfig_WithEnvFile_EnvTestIsolated tests LoadConfig with .env file in a completely isolated envtest environment
// This test ensures that .env file values are used without any interference from external environment variables
func TestLoadConfig_WithEnvFile_EnvTestIsolated(t *testing.T) {
	// Skip if envtest is not available
	if !testutil.IsEnvTestAvailable() {
		t.Skip("envtest not available, skipping isolated envtest test")
	}

	// Set up envtest environment to create an isolated testing context
	env, clientset, cleanup := testutil.SetupEnv(t)
	defer cleanup()

	// Reset Viper to clear any cached values
	viper.Reset()

	// Clear all relevant environment variables to ensure clean state
	originalPort := os.Getenv("PORT")
	originalLoggingLevel := os.Getenv("LOGGING_LEVEL")
	originalKubeconfig := os.Getenv("KUBECONFIG")
	originalNamespace := os.Getenv("NAMESPACE")
	originalInCluster := os.Getenv("IN_CLUSTER")

	// Unset all environment variables
	if err := os.Unsetenv("PORT"); err != nil {
		t.Fatalf("Failed to unset PORT env var: %v", err)
	}
	if err := os.Unsetenv("LOGGING_LEVEL"); err != nil {
		t.Fatalf("Failed to unset LOGGING_LEVEL env var: %v", err)
	}
	if err := os.Unsetenv("KUBECONFIG"); err != nil {
		t.Fatalf("Failed to unset KUBECONFIG env var: %v", err)
	}
	if err := os.Unsetenv("NAMESPACE"); err != nil {
		t.Fatalf("Failed to unset NAMESPACE env var: %v", err)
	}
	if err := os.Unsetenv("IN_CLUSTER"); err != nil {
		t.Fatalf("Failed to unset IN_CLUSTER env var: %v", err)
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
		if originalNamespace != "" {
			if err := os.Setenv("NAMESPACE", originalNamespace); err != nil {
				t.Errorf("Failed to restore NAMESPACE env var: %v", err)
			}
		}
		if originalInCluster != "" {
			if err := os.Setenv("IN_CLUSTER", originalInCluster); err != nil {
				t.Errorf("Failed to restore IN_CLUSTER env var: %v", err)
			}
		}
	}()

	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Create a test .env file with specific values
	envContent := `PORT=9090
LOGGING_LEVEL=debug
KUBECONFIG=/tmp/envtest.kubeconfig
NAMESPACE=default
IN_CLUSTER=false`

	envFile := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Load config from the test directory
	config, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify that .env file values are used (no environment variables to override them)
	expected := Config{
		Port:         "9090",                    // From .env file
		KUBECONFIG:   "/tmp/envtest.kubeconfig", // From .env file
		LoggingLevel: "debug",                   // From .env file
		Namespace:    "default",                 // From .env file
		InCluster:    false,                     // From .env file
	}

	if config != expected {
		t.Errorf("LoadConfig() = %v, want %v", config, expected)
	}

	// Verify that the kubeconfig file exists (created by envtest)
	if _, err := os.Stat("/tmp/envtest.kubeconfig"); err != nil {
		t.Errorf("Expected kubeconfig file to exist: %v", err)
	}

	// Test that the configuration works with the actual Kubernetes environment
	ctx := context.Background()
	deployments, err := clientset.AppsV1().Deployments(config.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Errorf("Failed to list deployments using config: %v", err)
	}

	// Verify that we can see the sample deployments created by SetupEnv
	if len(deployments.Items) < 2 {
		t.Errorf("Expected at least 2 deployments, got %d", len(deployments.Items))
	}

	// Verify that the environment is working correctly
	require.NotNil(t, env, "Environment should not be nil")
	require.NotNil(t, clientset, "Clientset should not be nil")

	t.Logf("Successfully loaded config from .env file in isolated envtest environment")
	t.Logf("Config: %+v", config)
	t.Logf("Found %d deployments in namespace %s", len(deployments.Items), config.Namespace)
}

// TestLoadConfig_PriorityOrder_EnvTest tests the complete priority order of configuration sources in envtest
// Priority order: CLI flags > Environment variables > .env file > Default values
func TestLoadConfig_PriorityOrder_EnvTest(t *testing.T) {
	// Skip if envtest is not available
	if !testutil.IsEnvTestAvailable() {
		t.Skip("envtest not available, skipping priority order test")
	}

	// Set up envtest environment
	env, clientset, cleanup := testutil.SetupEnv(t)
	defer cleanup()

	// Reset Viper to clear any cached values
	viper.Reset()

	// Clear all relevant environment variables to start with clean state
	originalPort := os.Getenv("PORT")
	originalLoggingLevel := os.Getenv("LOGGING_LEVEL")
	originalKubeconfig := os.Getenv("KUBECONFIG")
	originalNamespace := os.Getenv("NAMESPACE")
	originalInCluster := os.Getenv("IN_CLUSTER")

	// Unset all environment variables
	if err := os.Unsetenv("PORT"); err != nil {
		t.Fatalf("Failed to unset PORT env var: %v", err)
	}
	if err := os.Unsetenv("LOGGING_LEVEL"); err != nil {
		t.Fatalf("Failed to unset LOGGING_LEVEL env var: %v", err)
	}
	if err := os.Unsetenv("KUBECONFIG"); err != nil {
		t.Fatalf("Failed to unset KUBECONFIG env var: %v", err)
	}
	if err := os.Unsetenv("NAMESPACE"); err != nil {
		t.Fatalf("Failed to unset NAMESPACE env var: %v", err)
	}
	if err := os.Unsetenv("IN_CLUSTER"); err != nil {
		t.Fatalf("Failed to unset IN_CLUSTER env var: %v", err)
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
		if originalNamespace != "" {
			if err := os.Setenv("NAMESPACE", originalNamespace); err != nil {
				t.Errorf("Failed to restore NAMESPACE env var: %v", err)
			}
		}
		if originalInCluster != "" {
			if err := os.Setenv("IN_CLUSTER", originalInCluster); err != nil {
				t.Errorf("Failed to restore IN_CLUSTER env var: %v", err)
			}
		}
	}()

	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Test Case 1: Only .env file (should use .env values, fallback to defaults)
	t.Run("Only .env file", func(t *testing.T) {
		viper.Reset()

		// Create .env file with some values
		envContent := `PORT=9090
LOGGING_LEVEL=debug
KUBECONFIG=/test/kube/config
NAMESPACE=test-namespace
IN_CLUSTER=true`

		envFile := filepath.Join(tempDir, ".env")
		if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
			t.Fatalf("Failed to create test .env file: %v", err)
		}

		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		expected := Config{
			Port:         "9090",              // From .env file
			KUBECONFIG:   "/test/kube/config", // From .env file
			LoggingLevel: "debug",             // From .env file
			Namespace:    "test-namespace",    // From .env file
			InCluster:    true,                // From .env file
		}

		if config != expected {
			t.Errorf("LoadConfig() = %v, want %v", config, expected)
		}
	})

	// Test Case 2: Environment variables override .env file
	t.Run("Environment variables override .env file", func(t *testing.T) {
		viper.Reset()

		// Create .env file
		envContent := `PORT=9090
LOGGING_LEVEL=debug
KUBECONFIG=/test/kube/config
NAMESPACE=test-namespace
IN_CLUSTER=true`

		envFile := filepath.Join(tempDir, ".env")
		if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
			t.Fatalf("Failed to create test .env file: %v", err)
		}

		// Set environment variables (should override .env file)
		if err := os.Setenv("PORT", "7070"); err != nil {
			t.Fatalf("Failed to set PORT env var: %v", err)
		}
		if err := os.Setenv("LOGGING_LEVEL", "warn"); err != nil {
			t.Fatalf("Failed to set LOGGING_LEVEL env var: %v", err)
		}
		if err := os.Setenv("KUBECONFIG", "/env/kube/config"); err != nil {
			t.Fatalf("Failed to set KUBECONFIG env var: %v", err)
		}
		if err := os.Setenv("NAMESPACE", "env-namespace"); err != nil {
			t.Fatalf("Failed to set NAMESPACE env var: %v", err)
		}
		if err := os.Setenv("IN_CLUSTER", "false"); err != nil {
			t.Fatalf("Failed to set IN_CLUSTER env var: %v", err)
		}

		// Clean up environment variables after this test case
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
			if err := os.Unsetenv("NAMESPACE"); err != nil {
				t.Errorf("Failed to unset NAMESPACE env var: %v", err)
			}
			if err := os.Unsetenv("IN_CLUSTER"); err != nil {
				t.Errorf("Failed to unset IN_CLUSTER env var: %v", err)
			}
		}()

		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		expected := Config{
			Port:         "7070",             // From environment variable
			KUBECONFIG:   "/env/kube/config", // From environment variable
			LoggingLevel: "warn",             // From environment variable
			Namespace:    "env-namespace",    // From environment variable
			InCluster:    false,              // From environment variable
		}

		if config != expected {
			t.Errorf("LoadConfig() = %v, want %v", config, expected)
		}
	})

	// Test Case 3: Partial environment variables (some from env, some from .env, some defaults)
	t.Run("Partial environment variables", func(t *testing.T) {
		viper.Reset()

		// Create .env file with some values
		envContent := `PORT=9090
LOGGING_LEVEL=debug
KUBECONFIG=/test/kube/config
NAMESPACE=test-namespace
IN_CLUSTER=true`

		envFile := filepath.Join(tempDir, ".env")
		if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
			t.Fatalf("Failed to create test .env file: %v", err)
		}

		// Set only some environment variables
		if err := os.Setenv("PORT", "7070"); err != nil {
			t.Fatalf("Failed to set PORT env var: %v", err)
		}
		if err := os.Setenv("NAMESPACE", "env-namespace"); err != nil {
			t.Fatalf("Failed to set NAMESPACE env var: %v", err)
		}

		// Clean up environment variables after this test case
		defer func() {
			if err := os.Unsetenv("PORT"); err != nil {
				t.Errorf("Failed to unset PORT env var: %v", err)
			}
			if err := os.Unsetenv("NAMESPACE"); err != nil {
				t.Errorf("Failed to unset NAMESPACE env var: %v", err)
			}
		}()

		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		expected := Config{
			Port:         "7070",              // From environment variable
			KUBECONFIG:   "/test/kube/config", // From .env file
			LoggingLevel: "debug",             // From .env file
			Namespace:    "env-namespace",     // From environment variable
			InCluster:    true,                // From .env file
		}

		if config != expected {
			t.Errorf("LoadConfig() = %v, want %v", config, expected)
		}
	})

	// Test Case 4: No .env file, no environment variables (should use defaults)
	t.Run("Default values only", func(t *testing.T) {
		viper.Reset()

		// Create empty directory (no .env file)
		emptyDir := t.TempDir()

		config, err := LoadConfig(emptyDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		expected := Config{
			Port:         "8080",           // Default value
			KUBECONFIG:   "~/.kube/config", // Default value
			LoggingLevel: "info",           // Default value
			Namespace:    "default",        // Default value
			InCluster:    false,            // Default value
		}

		if config != expected {
			t.Errorf("LoadConfig() = %v, want %v", config, expected)
		}
	})

	// Test Case 5: Empty .env file (should use defaults)
	t.Run("Empty .env file", func(t *testing.T) {
		viper.Reset()

		// Create empty .env file
		envFile := filepath.Join(tempDir, ".env")
		if err := os.WriteFile(envFile, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create empty .env file: %v", err)
		}

		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		expected := Config{
			Port:         "8080",           // Default value
			KUBECONFIG:   "~/.kube/config", // Default value
			LoggingLevel: "info",           // Default value
			Namespace:    "default",        // Default value
			InCluster:    false,            // Default value
		}

		if config != expected {
			t.Errorf("LoadConfig() = %v, want %v", config, expected)
		}
	})

	// Test Case 6: Verify that the configuration works with actual Kubernetes environment
	t.Run("Integration with Kubernetes", func(t *testing.T) {
		viper.Reset()

		// Create .env file with realistic Kubernetes settings
		envContent := `PORT=9090
LOGGING_LEVEL=debug
KUBECONFIG=/tmp/envtest.kubeconfig
NAMESPACE=default
IN_CLUSTER=false`

		envFile := filepath.Join(tempDir, ".env")
		if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
			t.Fatalf("Failed to create test .env file: %v", err)
		}

		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Verify that the kubeconfig file exists (created by envtest)
		if _, err := os.Stat("/tmp/envtest.kubeconfig"); err != nil {
			t.Errorf("Expected kubeconfig file to exist: %v", err)
		}

		// Test that the configuration works with the actual Kubernetes environment
		ctx := context.Background()
		deployments, err := clientset.AppsV1().Deployments(config.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Errorf("Failed to list deployments using config: %v", err)
		}

		// Verify that we can see the sample deployments created by SetupEnv
		if len(deployments.Items) < 2 {
			t.Errorf("Expected at least 2 deployments, got %d", len(deployments.Items))
		}

		t.Logf("Successfully tested configuration integration with Kubernetes")
		t.Logf("Config: %+v", config)
		t.Logf("Found %d deployments in namespace %s", len(deployments.Items), config.Namespace)
	})

	// Verify that the environment is working correctly
	require.NotNil(t, env, "Environment should not be nil")
	require.NotNil(t, clientset, "Clientset should not be nil")

	t.Logf("Completed comprehensive priority order testing in envtest environment")
}

// TestLoadConfig_CLIFlagsPriority tests that CLI flags have the highest priority
// This test simulates CLI flag behavior by directly setting Viper values
func TestLoadConfig_CLIFlagsPriority(t *testing.T) {
	// Skip if envtest is not available
	if !testutil.IsEnvTestAvailable() {
		t.Skip("envtest not available, skipping CLI flags priority test")
	}

	// Set up envtest environment
	env, clientset, cleanup := testutil.SetupEnv(t)
	defer cleanup()

	// Reset Viper to clear any cached values
	viper.Reset()

	// Clear all relevant environment variables to start with clean state
	originalPort := os.Getenv("PORT")
	originalLoggingLevel := os.Getenv("LOGGING_LEVEL")
	originalKubeconfig := os.Getenv("KUBECONFIG")
	originalNamespace := os.Getenv("NAMESPACE")
	originalInCluster := os.Getenv("IN_CLUSTER")

	// Unset all environment variables
	if err := os.Unsetenv("PORT"); err != nil {
		t.Fatalf("Failed to unset PORT env var: %v", err)
	}
	if err := os.Unsetenv("LOGGING_LEVEL"); err != nil {
		t.Fatalf("Failed to unset LOGGING_LEVEL env var: %v", err)
	}
	if err := os.Unsetenv("KUBECONFIG"); err != nil {
		t.Fatalf("Failed to unset KUBECONFIG env var: %v", err)
	}
	if err := os.Unsetenv("NAMESPACE"); err != nil {
		t.Fatalf("Failed to unset NAMESPACE env var: %v", err)
	}
	if err := os.Unsetenv("IN_CLUSTER"); err != nil {
		t.Fatalf("Failed to unset IN_CLUSTER env var: %v", err)
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
		if originalNamespace != "" {
			if err := os.Setenv("NAMESPACE", originalNamespace); err != nil {
				t.Errorf("Failed to restore NAMESPACE env var: %v", err)
			}
		}
		if originalInCluster != "" {
			if err := os.Setenv("IN_CLUSTER", originalInCluster); err != nil {
				t.Errorf("Failed to restore IN_CLUSTER env var: %v", err)
			}
		}
	}()

	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Test Case 1: CLI flags override environment variables and .env file
	t.Run("CLI flags override everything", func(t *testing.T) {
		viper.Reset()

		// Create .env file with some values
		envContent := `PORT=9090
LOGGING_LEVEL=debug
KUBECONFIG=/test/kube/config
NAMESPACE=test-namespace
IN_CLUSTER=true`

		envFile := filepath.Join(tempDir, ".env")
		if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
			t.Fatalf("Failed to create test .env file: %v", err)
		}

		// Set environment variables (should be overridden by CLI flags)
		if err := os.Setenv("PORT", "7070"); err != nil {
			t.Fatalf("Failed to set PORT env var: %v", err)
		}
		if err := os.Setenv("LOGGING_LEVEL", "warn"); err != nil {
			t.Fatalf("Failed to set LOGGING_LEVEL env var: %v", err)
		}
		if err := os.Setenv("KUBECONFIG", "/env/kube/config"); err != nil {
			t.Fatalf("Failed to set KUBECONFIG env var: %v", err)
		}
		if err := os.Setenv("NAMESPACE", "env-namespace"); err != nil {
			t.Fatalf("Failed to set NAMESPACE env var: %v", err)
		}
		if err := os.Setenv("IN_CLUSTER", "false"); err != nil {
			t.Fatalf("Failed to set IN_CLUSTER env var: %v", err)
		}

		// Clean up environment variables after this test case
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
			if err := os.Unsetenv("NAMESPACE"); err != nil {
				t.Errorf("Failed to unset NAMESPACE env var: %v", err)
			}
			if err := os.Unsetenv("IN_CLUSTER"); err != nil {
				t.Errorf("Failed to unset IN_CLUSTER env var: %v", err)
			}
		}()

		// Simulate CLI flags by setting Viper values directly
		// This is how cobra would set the values when CLI flags are used
		viper.Set("PORT", "8080")                   // CLI flag value
		viper.Set("LOGGING_LEVEL", "error")         // CLI flag value
		viper.Set("KUBECONFIG", "/cli/kube/config") // CLI flag value
		viper.Set("NAMESPACE", "cli-namespace")     // CLI flag value
		viper.Set("IN_CLUSTER", "true")             // CLI flag value

		// Load config from the test directory
		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Verify that CLI flags have highest priority
		expected := Config{
			Port:         "8080",             // From CLI flag
			KUBECONFIG:   "/cli/kube/config", // From CLI flag
			LoggingLevel: "error",            // From CLI flag
			Namespace:    "cli-namespace",    // From CLI flag
			InCluster:    true,               // From CLI flag
		}

		if config != expected {
			t.Errorf("LoadConfig() = %v, want %v", config, expected)
		}
	})

	// Test Case 2: Partial CLI flags (some from CLI, some from env, some from .env, some defaults)
	t.Run("Partial CLI flags", func(t *testing.T) {
		viper.Reset()

		// Create .env file with some values
		envContent := `PORT=9090
LOGGING_LEVEL=debug
KUBECONFIG=/test/kube/config
NAMESPACE=test-namespace
IN_CLUSTER=true`

		envFile := filepath.Join(tempDir, ".env")
		if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
			t.Fatalf("Failed to create test .env file: %v", err)
		}

		// Set only some environment variables
		if err := os.Setenv("PORT", "7070"); err != nil {
			t.Fatalf("Failed to set PORT env var: %v", err)
		}
		if err := os.Setenv("NAMESPACE", "env-namespace"); err != nil {
			t.Fatalf("Failed to set NAMESPACE env var: %v", err)
		}

		// Clean up environment variables after this test case
		defer func() {
			if err := os.Unsetenv("PORT"); err != nil {
				t.Errorf("Failed to unset PORT env var: %v", err)
			}
			if err := os.Unsetenv("NAMESPACE"); err != nil {
				t.Errorf("Failed to unset NAMESPACE env var: %v", err)
			}
		}()

		// Set only some CLI flags
		viper.Set("PORT", "8080")           // CLI flag value
		viper.Set("LOGGING_LEVEL", "error") // CLI flag value

		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		expected := Config{
			Port:         "8080",              // From CLI flag
			KUBECONFIG:   "/test/kube/config", // From .env file
			LoggingLevel: "error",             // From CLI flag
			Namespace:    "env-namespace",     // From environment variable
			InCluster:    true,                // From .env file
		}

		if config != expected {
			t.Errorf("LoadConfig() = %v, want %v", config, expected)
		}
	})

	// Test Case 3: Integration with Kubernetes using CLI flags
	t.Run("CLI flags with Kubernetes integration", func(t *testing.T) {
		viper.Reset()

		// Create .env file with realistic Kubernetes settings
		envContent := `PORT=9090
LOGGING_LEVEL=debug
KUBECONFIG=/tmp/envtest.kubeconfig
NAMESPACE=default
IN_CLUSTER=false`

		envFile := filepath.Join(tempDir, ".env")
		if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
			t.Fatalf("Failed to create test .env file: %v", err)
		}

		// Set CLI flags that override .env file
		viper.Set("PORT", "8080")          // CLI flag value
		viper.Set("LOGGING_LEVEL", "info") // CLI flag value
		viper.Set("NAMESPACE", "default")  // CLI flag value (same as .env)

		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Verify that the kubeconfig file exists (created by envtest)
		if _, err := os.Stat("/tmp/envtest.kubeconfig"); err != nil {
			t.Errorf("Expected kubeconfig file to exist: %v", err)
		}

		// Test that the configuration works with the actual Kubernetes environment
		ctx := context.Background()
		deployments, err := clientset.AppsV1().Deployments(config.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Errorf("Failed to list deployments using config: %v", err)
		}

		// Verify that we can see the sample deployments created by SetupEnv
		if len(deployments.Items) < 2 {
			t.Errorf("Expected at least 2 deployments, got %d", len(deployments.Items))
		}

		expected := Config{
			Port:         "8080",                    // From CLI flag
			KUBECONFIG:   "/tmp/envtest.kubeconfig", // From .env file
			LoggingLevel: "info",                    // From CLI flag
			Namespace:    "default",                 // From CLI flag
			InCluster:    false,                     // From .env file
		}

		if config != expected {
			t.Errorf("LoadConfig() = %v, want %v", config, expected)
		}

		t.Logf("Successfully tested CLI flags with Kubernetes integration")
		t.Logf("Config: %+v", config)
		t.Logf("Found %d deployments in namespace %s", len(deployments.Items), config.Namespace)
	})

	// Verify that the environment is working correctly
	require.NotNil(t, env, "Environment should not be nil")
	require.NotNil(t, clientset, "Clientset should not be nil")

	t.Logf("Completed CLI flags priority testing in envtest environment")
}
