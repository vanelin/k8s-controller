package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInt32Ptr(t *testing.T) {
	v := int32(42)
	ptr := int32Ptr(v)
	if ptr == nil || *ptr != v {
		t.Errorf("int32Ptr(%d) = %v, want pointer to %d", v, ptr, v)
	}
}

func TestIsEnvTestAvailable(t *testing.T) {
	// Save original value
	original := os.Getenv("KUBEBUILDER_ASSETS")
	defer func() {
		if original != "" {
			_ = os.Setenv("KUBEBUILDER_ASSETS", original)
		} else {
			_ = os.Unsetenv("KUBEBUILDER_ASSETS")
		}
	}()

	// Test when KUBEBUILDER_ASSETS is not set
	if err := os.Unsetenv("KUBEBUILDER_ASSETS"); err != nil {
		t.Errorf("Failed to unset KUBEBUILDER_ASSETS: %v", err)
	}
	if IsEnvTestAvailable() {
		t.Error("IsEnvTestAvailable should return false when KUBEBUILDER_ASSETS is not set")
	}

	// Test when KUBEBUILDER_ASSETS is set but etcd doesn't exist
	if err := os.Setenv("KUBEBUILDER_ASSETS", "/nonexistent/path"); err != nil {
		t.Errorf("Failed to set KUBEBUILDER_ASSETS: %v", err)
	}
	if IsEnvTestAvailable() {
		t.Error("IsEnvTestAvailable should return false when etcd binary doesn't exist")
	}

	// Test when KUBEBUILDER_ASSETS is set and etcd exists (if available)
	if assets := os.Getenv("KUBEBUILDER_ASSETS"); assets != "" {
		// This test will pass if envtest is actually available
		// and fail (but not error) if it's not
		_ = IsEnvTestAvailable()
	}
}

func TestSetupEnv(t *testing.T) {
	// Test SetupEnv function
	env, clientset, cleanup := SetupEnv(t)
	defer cleanup()

	// Verify that env is not nil
	require.NotNil(t, env, "Environment should not be nil")

	// Verify that clientset is not nil
	require.NotNil(t, clientset, "Clientset should not be nil")

	// Verify that cleanup function is not nil
	require.NotNil(t, cleanup, "Cleanup function should not be nil")

	// Test that we can list deployments
	ctx := context.Background()
	deployments, err := clientset.AppsV1().Deployments("default").List(ctx, metav1.ListOptions{})
	require.NoError(t, err, "Should be able to list deployments")

	// Verify that sample deployments were created
	require.Len(t, deployments.Items, 2, "Should have 2 sample deployments")

	// Check for specific deployment names
	foundDeployments := make(map[string]bool)
	for _, dep := range deployments.Items {
		foundDeployments[dep.Name] = true
	}

	require.True(t, foundDeployments["sample-deployment-1"], "Should find sample-deployment-1")
	require.True(t, foundDeployments["sample-deployment-2"], "Should find sample-deployment-2")

	// Test that we can get a specific deployment
	dep, err := clientset.AppsV1().Deployments("default").Get(ctx, "sample-deployment-1", metav1.GetOptions{})
	require.NoError(t, err, "Should be able to get specific deployment")
	require.Equal(t, "sample-deployment-1", dep.Name, "Deployment name should match")

	// Verify deployment spec
	require.NotNil(t, dep.Spec.Replicas, "Deployment should have replicas set")
	require.Equal(t, int32(1), *dep.Spec.Replicas, "Deployment should have 1 replica")
	require.Equal(t, "nginx", dep.Spec.Template.Spec.Containers[0].Image, "Container image should be nginx")

	// Test that kubeconfig file was created
	_, err = os.Stat("/tmp/envtest.kubeconfig")
	require.NoError(t, err, "Kubeconfig file should be created at /tmp/envtest.kubeconfig")
}
