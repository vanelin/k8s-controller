package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/vanelin/k8s-controller/pkg/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// envSnapshot saves and restores environment variables for test isolation
func envSnapshot(t *testing.T, keys ...string) func() {
	t.Helper()
	originals := make(map[string]string, len(keys))
	for _, k := range keys {
		originals[k] = os.Getenv(k)
		_ = os.Unsetenv(k)
	}
	return func() {
		for _, k := range keys {
			if v, ok := originals[k]; ok && v != "" {
				_ = os.Setenv(k, v)
			} else {
				_ = os.Unsetenv(k)
			}
		}
	}
}

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
				Port:                    "8080",
				KUBECONFIG:              "~/.kube/config",
				LoggingLevel:            "info",
				Namespace:               "default",
				InCluster:               false,
				MetricPort:              "8081",
				EnableLeaderElection:    true,
				LeaderElectionNamespace: "default",
			},
		},
		{
			name: "partial config should set missing defaults",
			config: Config{
				Port: "9090",
			},
			expected: Config{
				Port:                    "9090",
				KUBECONFIG:              "~/.kube/config",
				LoggingLevel:            "info",
				Namespace:               "default",
				InCluster:               false,
				MetricPort:              "8081",
				EnableLeaderElection:    true,
				LeaderElectionNamespace: "default",
			},
		},
		{
			name: "full config should not change",
			config: Config{
				Port:                    "9090",
				KUBECONFIG:              "/custom/kube/config",
				LoggingLevel:            "debug",
				Namespace:               "custom-namespace",
				InCluster:               true,
				MetricPort:              "9091",
				EnableLeaderElection:    false,
				LeaderElectionNamespace: "custom-leader-namespace",
			},
			expected: Config{
				Port:                    "9090",
				KUBECONFIG:              "/custom/kube/config",
				LoggingLevel:            "debug",
				Namespace:               "custom-namespace",
				InCluster:               true,
				MetricPort:              "9091",
				EnableLeaderElection:    false,
				LeaderElectionNamespace: "custom-leader-namespace",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for each test to ensure clean state
			viper.Reset()

			// For the "full config should not change" test, we need to simulate
			// that the value was set via viper to prevent it from being overridden
			if tt.name == "full config should not change" {
				// Simulate that ENABLE_LEADER_ELECTION was set via viper
				viper.Set("ENABLE_LEADER_ELECTION", false)
			}

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
		Port:                    "8080",
		KUBECONFIG:              "~/.kube/config",
		LoggingLevel:            "info",
		Namespace:               "test-namespace",
		InCluster:               false,
		EnableLeaderElection:    false,
		LeaderElectionNamespace: "default",
	}

	// This test mainly ensures PrintConfig doesn't panic
	// In a real scenario, you might want to capture stdout and verify the output
	config.PrintConfig()
}

func TestLoadConfig_WithEnvFile(t *testing.T) {
	viper.Reset()
	// Use helper for environment isolation
	cleanup := envSnapshot(t,
		"PORT", "LOGGING_LEVEL", "KUBECONFIG", "NAMESPACE", "IN_CLUSTER", "METRIC_PORT", "ENABLE_LEADER_ELECTION", "LEADER_ELECTION_NAMESPACE",
	)
	defer cleanup()

	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Create a test .env file
	envContent := `PORT=9090
LOGGING_LEVEL=debug
KUBECONFIG=/test/kube/config
NAMESPACE=test-namespace
IN_CLUSTER=true
METRIC_PORT=9091
ENABLE_LEADER_ELECTION=false
LEADER_ELECTION_NAMESPACE=fromenvfile`

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
		Port:                    "9090",
		KUBECONFIG:              "/test/kube/config",
		LoggingLevel:            "debug",
		Namespace:               "test-namespace",
		InCluster:               true,
		MetricPort:              "9091",
		EnableLeaderElection:    false,
		LeaderElectionNamespace: "fromenvfile",
	}

	if config != expected {
		t.Errorf("LoadConfig() = %v, want %v", config, expected)
	}
}

func TestLoadConfig_WithEnvironmentVariables(t *testing.T) {
	cleanup := envSnapshot(t,
		"PORT", "LOGGING_LEVEL", "KUBECONFIG", "NAMESPACE", "IN_CLUSTER", "METRIC_PORT", "ENABLE_LEADER_ELECTION", "LEADER_ELECTION_NAMESPACE",
	)
	defer cleanup()

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
	if err := os.Setenv("METRIC_PORT", "7071"); err != nil {
		t.Fatalf("Failed to set METRIC_PORT env var: %v", err)
	}
	if err := os.Setenv("ENABLE_LEADER_ELECTION", "true"); err != nil {
		t.Fatalf("Failed to set ENABLE_LEADER_ELECTION env var: %v", err)
	}
	if err := os.Setenv("LEADER_ELECTION_NAMESPACE", "fromenv"); err != nil {
		t.Fatalf("Failed to set LEADER_ELECTION_NAMESPACE env var: %v", err)
	}

	// Load config (should use environment variables)
	config, err := LoadConfig("nonexistent/path")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify the loaded values
	expected := Config{
		Port:                    "7070",
		KUBECONFIG:              "/env/kube/config",
		LoggingLevel:            "warn",
		Namespace:               "env-namespace",
		InCluster:               true,
		MetricPort:              "7071",
		EnableLeaderElection:    true,
		LeaderElectionNamespace: "fromenv",
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
	originalMetricPort := os.Getenv("METRIC_PORT")
	originalLeaderElectionNamespace := os.Getenv("LEADER_ELECTION_NAMESPACE")

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
	if err := os.Unsetenv("METRIC_PORT"); err != nil {
		t.Fatalf("Failed to unset METRIC_PORT env var: %v", err)
	}
	if err := os.Unsetenv("LEADER_ELECTION_NAMESPACE"); err != nil {
		t.Fatalf("Failed to unset LEADER_ELECTION_NAMESPACE env var: %v", err)
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
		if originalMetricPort != "" {
			if err := os.Setenv("METRIC_PORT", originalMetricPort); err != nil {
				t.Errorf("Failed to restore METRIC_PORT env var: %v", err)
			}
		}
		if originalLeaderElectionNamespace != "" {
			if err := os.Setenv("LEADER_ELECTION_NAMESPACE", originalLeaderElectionNamespace); err != nil {
				t.Errorf("Failed to restore LEADER_ELECTION_NAMESPACE env var: %v", err)
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
	if config.MetricPort != "8081" {
		t.Errorf("Expected default METRIC_PORT=8081, got %s", config.MetricPort)
	}
	if config.EnableLeaderElection != true {
		t.Errorf("Expected default ENABLE_LEADER_ELECTION=true, got %t", config.EnableLeaderElection)
	}
	if config.LeaderElectionNamespace != "default" {
		t.Errorf("Expected default LeaderElectionNamespace=default, got %s", config.LeaderElectionNamespace)
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
	if config.MetricPort != "8081" {
		t.Errorf("Expected default METRIC_PORT=8081, got %s", config.MetricPort)
	}
	if config.EnableLeaderElection != true {
		t.Errorf("Expected default ENABLE_LEADER_ELECTION=true, got %t", config.EnableLeaderElection)
	}
	if config.LeaderElectionNamespace != "default" {
		t.Errorf("Expected default LeaderElectionNamespace=default, got %s", config.LeaderElectionNamespace)
	}
}

// TestLoadConfig_EnvOverridesEnvFile explicitly tests that environment variables override .env file values
func TestLoadConfig_EnvOverridesEnvFile(t *testing.T) {
	viper.Reset()
	// Reset Viper to clear any cached values
	originalPort := os.Getenv("PORT")
	originalLoggingLevel := os.Getenv("LOGGING_LEVEL")
	originalKubeconfig := os.Getenv("KUBECONFIG")
	originalNamespace := os.Getenv("NAMESPACE")
	originalInCluster := os.Getenv("IN_CLUSTER")
	originalMetricPort := os.Getenv("METRIC_PORT")
	originalEnableLeaderElection := os.Getenv("ENABLE_LEADER_ELECTION")
	originalLeaderElectionNamespace := os.Getenv("LEADER_ELECTION_NAMESPACE")

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
	if err := os.Unsetenv("METRIC_PORT"); err != nil {
		t.Fatalf("Failed to unset METRIC_PORT env var: %v", err)
	}
	if err := os.Unsetenv("ENABLE_LEADER_ELECTION"); err != nil {
		t.Fatalf("Failed to unset ENABLE_LEADER_ELECTION env var: %v", err)
	}
	if err := os.Unsetenv("LEADER_ELECTION_NAMESPACE"); err != nil {
		t.Fatalf("Failed to unset LEADER_ELECTION_NAMESPACE env var: %v", err)
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
		if originalMetricPort != "" {
			if err := os.Setenv("METRIC_PORT", originalMetricPort); err != nil {
				t.Errorf("Failed to restore METRIC_PORT env var: %v", err)
			}
		}
		if originalEnableLeaderElection != "" {
			if err := os.Setenv("ENABLE_LEADER_ELECTION", originalEnableLeaderElection); err != nil {
				t.Errorf("Failed to restore ENABLE_LEADER_ELECTION env var: %v", err)
			}
		}
		if originalLeaderElectionNamespace != "" {
			if err := os.Setenv("LEADER_ELECTION_NAMESPACE", originalLeaderElectionNamespace); err != nil {
				t.Errorf("Failed to restore LEADER_ELECTION_NAMESPACE env var: %v", err)
			}
		}
	}()

	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Create a .env file with one set of values
	envContent := `PORT=9090
LOGGING_LEVEL=debug
KUBECONFIG=/test/kube/config
NAMESPACE=test-namespace
IN_CLUSTER=false
METRIC_PORT=9091
ENABLE_LEADER_ELECTION=false
LEADER_ELECTION_NAMESPACE=fromenvfile`

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
	if err := os.Setenv("METRIC_PORT", "8031"); err != nil {
		t.Fatalf("Failed to set METRIC_PORT env var: %v", err)
	}
	if err := os.Setenv("ENABLE_LEADER_ELECTION", "true"); err != nil {
		t.Fatalf("Failed to set ENABLE_LEADER_ELECTION env var: %v", err)
	}
	if err := os.Setenv("LEADER_ELECTION_NAMESPACE", "fromenv"); err != nil {
		t.Fatalf("Failed to set LEADER_ELECTION_NAMESPACE env var: %v", err)
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
		if err := os.Unsetenv("METRIC_PORT"); err != nil {
			t.Errorf("Failed to unset METRIC_PORT env var: %v", err)
		}
		if err := os.Unsetenv("ENABLE_LEADER_ELECTION"); err != nil {
			t.Errorf("Failed to unset ENABLE_LEADER_ELECTION env var: %v", err)
		}
		if err := os.Unsetenv("LEADER_ELECTION_NAMESPACE"); err != nil {
			t.Errorf("Failed to unset LEADER_ELECTION_NAMESPACE env var: %v", err)
		}
	}()

	// Load config from the test directory
	config, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify that environment variables override .env file values
	expected := Config{
		Port:                    "8030",
		KUBECONFIG:              "/env/kube/config",
		LoggingLevel:            "warn",
		Namespace:               "env-namespace",
		InCluster:               true,
		MetricPort:              "8031",
		EnableLeaderElection:    true,
		LeaderElectionNamespace: "fromenv",
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
IN_CLUSTER=false
METRIC_PORT=9091
ENABLE_LEADER_ELECTION=false
LEADER_ELECTION_NAMESPACE=test-leader-election-namespace`

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
		Port:                    "9090",
		KUBECONFIG:              "/tmp/envtest.kubeconfig",
		LoggingLevel:            "debug",
		Namespace:               "default",
		InCluster:               false,
		MetricPort:              "9091",
		EnableLeaderElection:    false,
		LeaderElectionNamespace: "test-leader-election-namespace",
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
			Port:                    "9090",
			KUBECONFIG:              "/test/kube/config",
			LoggingLevel:            "debug",
			Namespace:               "test-namespace",
			InCluster:               true,
			MetricPort:              "8081",
			EnableLeaderElection:    true,
			LeaderElectionNamespace: "default",
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
			Port:                    "7070",
			KUBECONFIG:              "/env/kube/config",
			LoggingLevel:            "warn",
			Namespace:               "env-namespace",
			InCluster:               false,
			MetricPort:              "8081",
			EnableLeaderElection:    true,
			LeaderElectionNamespace: "default",
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
			Port:                    "7070",
			KUBECONFIG:              "/test/kube/config",
			LoggingLevel:            "debug",
			Namespace:               "env-namespace",
			InCluster:               true,
			MetricPort:              "8081",
			EnableLeaderElection:    true,
			LeaderElectionNamespace: "default",
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
			Port:                    "8080",
			KUBECONFIG:              "~/.kube/config",
			LoggingLevel:            "info",
			Namespace:               "default",
			InCluster:               false,
			MetricPort:              "8081",
			EnableLeaderElection:    true,
			LeaderElectionNamespace: "default",
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
			Port:                    "8080",
			KUBECONFIG:              "~/.kube/config",
			LoggingLevel:            "info",
			Namespace:               "default",
			InCluster:               false,
			MetricPort:              "8081",
			EnableLeaderElection:    true,
			LeaderElectionNamespace: "default",
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
IN_CLUSTER=true
ENABLE_LEADER_ELECTION=false
LEADER_ELECTION_NAMESPACE=fromenvfile`

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
		viper.Set("PORT", "8080")
		viper.Set("LOGGING_LEVEL", "error")
		viper.Set("KUBECONFIG", "/cli/kube/config")
		viper.Set("NAMESPACE", "cli-namespace")
		viper.Set("IN_CLUSTER", "true")
		viper.Set("ENABLE_LEADER_ELECTION", false)
		viper.Set("LEADER_ELECTION_NAMESPACE", "fromcli")

		// Load config from the test directory
		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Verify that CLI flags have highest priority
		expected := Config{
			Port:                    "8080",
			KUBECONFIG:              "/cli/kube/config",
			LoggingLevel:            "error",
			Namespace:               "cli-namespace",
			InCluster:               true,
			MetricPort:              "8081",
			EnableLeaderElection:    false,
			LeaderElectionNamespace: "fromcli",
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
		viper.Set("PORT", "8080")
		viper.Set("LOGGING_LEVEL", "error")

		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		expected := Config{
			Port:                    "8080",
			KUBECONFIG:              "/test/kube/config",
			LoggingLevel:            "error",
			Namespace:               "env-namespace",
			InCluster:               true,
			MetricPort:              "8081",
			EnableLeaderElection:    true,
			LeaderElectionNamespace: "default",
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
		viper.Set("PORT", "8080")
		viper.Set("LOGGING_LEVEL", "info")
		viper.Set("NAMESPACE", "default")

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
			Port:                    "8080",
			KUBECONFIG:              "/tmp/envtest.kubeconfig",
			LoggingLevel:            "info",
			Namespace:               "default",
			InCluster:               false,
			MetricPort:              "8081",
			EnableLeaderElection:    true,
			LeaderElectionNamespace: "default",
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
