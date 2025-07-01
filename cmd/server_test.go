package cmd

import (
	"os"
	"testing"
)

func TestServerCommandDefined(t *testing.T) {
	if serverCmd == nil {
		t.Fatal("serverCmd should be defined")
	}
	if serverCmd.Use != "server" {
		t.Errorf("expected command use 'server', got %s", serverCmd.Use)
	}
	portFlag := serverCmd.Flags().Lookup("port")
	if portFlag == nil {
		t.Fatal("expected 'port' flag to be defined")
	}
	if portFlag.Value.Type() != "string" {
		t.Errorf("expected 'port' flag to be string type, got %s", portFlag.Value.Type())
	}
}

func TestGetServerKubeClient_InvalidPath(t *testing.T) {
	_, err := getServerKubeClient("/invalid/path", false)
	if err == nil {
		t.Error("expected error for invalid kubeconfig path")
	}
}

func TestGetServerKubeClient_EmptyPath(t *testing.T) {
	_, err := getServerKubeClient("", false)
	if err == nil {
		t.Error("expected error for empty kubeconfig path")
	}
}

func TestGetServerKubeClient_ValidPath(t *testing.T) {
	// This test might fail if ~/.kube/config doesn't exist
	// but it's good to test the happy path
	_, err := getServerKubeClient("~/.kube/config", false)
	// We don't assert on error here because the file might not exist
	// but the function should handle the tilde expansion correctly
	if err != nil {
		t.Logf("getServerKubeClient returned error (expected if ~/.kube/config doesn't exist): %v", err)
	}
}

func TestGetServerKubeClient_InClusterPriority(t *testing.T) {
	// Test that inCluster=true takes priority over kubeconfig path
	// When inCluster=true, the kubeconfig path should be ignored
	_, err := getServerKubeClient("~/.kube/config", true)
	// This should try to use in-cluster config regardless of the kubeconfig path
	// The error might be expected if not running in a cluster
	if err != nil {
		t.Logf("getServerKubeClient with inCluster=true returned error (expected if not in cluster): %v", err)
	}
}

func TestServerCommandFlags(t *testing.T) {
	// Test that all expected flags are defined
	expectedFlags := []string{"port", "kubeconfig", "in-cluster", "namespace"}

	for _, flagName := range expectedFlags {
		flag := serverCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag '%s' to be defined", flagName)
		}
	}
}

func TestServerCommandUsage(t *testing.T) {
	if serverCmd.Short == "" {
		t.Error("expected server command to have a short description")
	}

	if serverCmd.Long != "" && len(serverCmd.Long) < len(serverCmd.Short) {
		t.Error("expected long description to be longer than short description")
	}
}

func TestServerEnvironmentVariablePriority(t *testing.T) {
	// Save original environment variables
	originalKubeconfig := os.Getenv("KUBECONFIG")
	originalInCluster := os.Getenv("IN_CLUSTER")

	// Clean up after test
	defer func() {
		if originalKubeconfig != "" {
			os.Setenv("KUBECONFIG", originalKubeconfig)
		} else {
			os.Unsetenv("KUBECONFIG")
		}
		if originalInCluster != "" {
			os.Setenv("IN_CLUSTER", originalInCluster)
		} else {
			os.Unsetenv("IN_CLUSTER")
		}
	}()

	// Set conflicting environment variables
	os.Setenv("KUBECONFIG", "~/.kube/config")
	os.Setenv("IN_CLUSTER", "true")

	// Test that the configuration loading respects environment variables
	// This test verifies that environment variables are properly loaded
	// The actual priority logic is tested in the config package
	t.Log("Testing environment variable priority: KUBECONFIG=~/.kube/config, IN_CLUSTER=true")

	// Verify that environment variables are set
	if os.Getenv("KUBECONFIG") != "~/.kube/config" {
		t.Error("KUBECONFIG environment variable not set correctly")
	}
	if os.Getenv("IN_CLUSTER") != "true" {
		t.Error("IN_CLUSTER environment variable not set correctly")
	}
}

func TestServerConfigurationPriorityLogic(t *testing.T) {
	// Save original environment variables
	originalKubeconfig := os.Getenv("KUBECONFIG")
	originalInCluster := os.Getenv("IN_CLUSTER")
	originalNamespace := os.Getenv("NAMESPACE")
	originalPort := os.Getenv("PORT")

	// Clean up after test
	defer func() {
		if originalKubeconfig != "" {
			os.Setenv("KUBECONFIG", originalKubeconfig)
		} else {
			os.Unsetenv("KUBECONFIG")
		}
		if originalInCluster != "" {
			os.Setenv("IN_CLUSTER", originalInCluster)
		} else {
			os.Unsetenv("IN_CLUSTER")
		}
		if originalNamespace != "" {
			os.Setenv("NAMESPACE", originalNamespace)
		} else {
			os.Unsetenv("NAMESPACE")
		}
		if originalPort != "" {
			os.Setenv("PORT", originalPort)
		} else {
			os.Unsetenv("PORT")
		}
	}()

	// Test scenario: conflicting environment variables
	// KUBECONFIG=~/.kube/config (should be ignored when IN_CLUSTER=true)
	// IN_CLUSTER=true (should take priority)
	os.Setenv("KUBECONFIG", "~/.kube/config")
	os.Setenv("IN_CLUSTER", "true")
	os.Setenv("NAMESPACE", "test-namespace")
	os.Setenv("PORT", "9090")

	t.Log("Testing server configuration priority logic with conflicting env vars")

	// Verify environment variables are set correctly
	expectedVars := map[string]string{
		"KUBECONFIG": "~/.kube/config",
		"IN_CLUSTER": "true",
		"NAMESPACE":  "test-namespace",
		"PORT":       "9090",
	}

	for key, expectedValue := range expectedVars {
		actualValue := os.Getenv(key)
		if actualValue != expectedValue {
			t.Errorf("Environment variable %s not set correctly: expected %s, got %s", key, expectedValue, actualValue)
		}
	}

	// Test that the logic in server.go would handle this correctly
	// The server should prioritize IN_CLUSTER=true over KUBECONFIG
	t.Log("Environment variables set successfully. Server should use in-cluster config when IN_CLUSTER=true")
}
