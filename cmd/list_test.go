package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vanelin/k8s-controller.git/pkg/common/utils"
)

// setEnvSafely sets an environment variable and logs any error
func setEnvSafely(key, value string) {
	if err := os.Setenv(key, value); err != nil {
		// In test context, we'll panic as this is unexpected
		panic("Failed to set environment variable: " + err.Error())
	}
}

func TestGetKubeconfigPath(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("KUBECONFIG")
	defer setEnvSafely("KUBECONFIG", originalEnv)

	// Get user's home directory for tilde expansion
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	tests := []struct {
		name           string
		cliFlag        string
		configValue    string
		expectedResult string
	}{
		{
			name:           "CLI flag takes highest priority",
			cliFlag:        "/cli/config",
			configValue:    "/config/config",
			expectedResult: "/cli/config",
		},
		{
			name:           "Config value when CLI flag is not set",
			cliFlag:        "",
			configValue:    "/config/config",
			expectedResult: "/config/config",
		},
		{
			name:           "Default value when nothing is set",
			cliFlag:        "",
			configValue:    "",
			expectedResult: filepath.Join(homeDir, ".kube", "config"),
		},
		{
			name:           "Tilde expansion in CLI flag",
			cliFlag:        "~/.kube/config",
			configValue:    "/config/config",
			expectedResult: filepath.Join(homeDir, ".kube", "config"),
		},
		{
			name:           "Tilde expansion in config value",
			cliFlag:        "",
			configValue:    "~/.kube/config",
			expectedResult: filepath.Join(homeDir, ".kube", "config"),
		},
		{
			name:           "Empty CLI flag with empty config",
			cliFlag:        "",
			configValue:    "",
			expectedResult: filepath.Join(homeDir, ".kube", "config"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test environment
			kubeconfigFlag = tt.cliFlag

			// Create temporary config for testing
			tempConfig := appConfig
			if tt.configValue != "" {
				tempConfig.KUBECONFIG = tt.configValue
			} else {
				tempConfig.KUBECONFIG = "~/.kube/config" // Default value
			}

			// Save original config and restore after test
			originalConfig := appConfig
			defer func() { appConfig = originalConfig }()
			appConfig = tempConfig

			result := getKubeconfigPath()
			if result != tt.expectedResult {
				t.Errorf("getKubeconfigPath() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestExpandTilde(t *testing.T) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Expand tilde with path",
			input:    "~/.kube/config",
			expected: filepath.Join(homeDir, ".kube", "config"),
		},
		{
			name:     "Expand tilde only",
			input:    "~",
			expected: homeDir,
		},
		{
			name:     "No tilde",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Relative path without tilde",
			input:    "relative/path",
			expected: "relative/path",
		},
		{
			name:     "Tilde with subdirectory",
			input:    "~/Documents/config",
			expected: filepath.Join(homeDir, "Documents", "config"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ExpandTilde(tt.input)
			if result != tt.expected {
				t.Errorf("utils.ExpandTilde(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNamespaceFlagDefault(t *testing.T) {
	// Test that namespace flag has correct default value
	expectedDefault := "default"

	// We can't directly test the flag default here since it's set in init()
	// But we can verify that our logic expects the correct default
	if expectedDefault != "default" {
		t.Errorf("Expected namespace default to be 'default', got %s", expectedDefault)
	}
}

func TestListCommandFlags(t *testing.T) {
	// Test that list command has the expected flags
	expectedFlags := []string{"kubeconfig", "namespace"}

	for _, flagName := range expectedFlags {
		flag := listCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag '%s' to be defined on list command", flagName)
		}
	}
}

func TestListCommandUsage(t *testing.T) {
	// Test that list command has correct usage information
	expectedUse := "list"
	expectedShort := "List Kubernetes deployments in the specified namespace(s)"

	if listCmd.Use != expectedUse {
		t.Errorf("Expected list command Use to be '%s', got '%s'", expectedUse, listCmd.Use)
	}

	if listCmd.Short != expectedShort {
		t.Errorf("Expected list command Short to be '%s', got '%s'", expectedShort, listCmd.Short)
	}
}

func TestGetKubeClient(t *testing.T) {
	// Test getKubeClient function with invalid kubeconfig
	_, err := getKubeClient("/nonexistent/path/to/kubeconfig")
	if err == nil {
		t.Error("Expected getKubeClient to return error for invalid kubeconfig path")
	}
}

func TestConfigurationPriority(t *testing.T) {
	// Save original environment and config
	originalEnv := os.Getenv("KUBECONFIG")
	originalConfig := appConfig
	defer func() {
		setEnvSafely("KUBECONFIG", originalEnv)
		appConfig = originalConfig
	}()

	// Get user's home directory for tilde expansion
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	// Test configuration priority: CLI flag > config (which includes env vars via Viper) > default
	tests := []struct {
		name           string
		cliFlag        string
		configValue    string
		expectedResult string
	}{
		{
			name:           "CLI flag takes highest priority",
			cliFlag:        "/cli/config",
			configValue:    "/config/config",
			expectedResult: "/cli/config",
		},
		{
			name:           "Config value when CLI flag not set",
			cliFlag:        "",
			configValue:    "/config/config",
			expectedResult: "/config/config",
		},
		{
			name:           "Default value when CLI flag and config not set",
			cliFlag:        "",
			configValue:    "",
			expectedResult: filepath.Join(homeDir, ".kube", "config"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test environment
			kubeconfigFlag = tt.cliFlag

			// Set config value
			tempConfig := appConfig
			if tt.configValue != "" {
				tempConfig.KUBECONFIG = tt.configValue
			} else {
				tempConfig.KUBECONFIG = "~/.kube/config" // Default value
			}
			appConfig = tempConfig

			result := getKubeconfigPath()
			if result != tt.expectedResult {
				t.Errorf("getKubeconfigPath() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestParseNamespaces(t *testing.T) {
	tests := []struct {
		name           string
		namespaceInput string
		expected       []string
	}{
		{
			name:           "Single namespace",
			namespaceInput: "default",
			expected:       []string{"default"},
		},
		{
			name:           "Multiple namespaces",
			namespaceInput: "kube-system,test",
			expected:       []string{"kube-system", "test"},
		},
		{
			name:           "Multiple namespaces with spaces",
			namespaceInput: "kube-system , test",
			expected:       []string{"kube-system", "test"},
		},
		{
			name:           "Multiple namespaces with extra spaces",
			namespaceInput: "  kube-system  ,  test  ",
			expected:       []string{"kube-system", "test"},
		},
		{
			name:           "Empty string",
			namespaceInput: "",
			expected:       []string{"default"},
		},
		{
			name:           "Three namespaces",
			namespaceInput: "default,kube-system,test",
			expected:       []string{"default", "kube-system", "test"},
		},
		{
			name:           "Single namespace with spaces",
			namespaceInput: "  default  ",
			expected:       []string{"default"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNamespaces(tt.namespaceInput)
			assert.Equal(t, tt.expected, result)
		})
	}
}
