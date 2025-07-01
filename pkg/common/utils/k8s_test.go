package utils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCheckNamespace(t *testing.T) {
	// Create a fake Kubernetes client
	fakeClient := fake.NewSimpleClientset()

	// Create a test namespace
	testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	_, err := fakeClient.CoreV1().Namespaces().Create(context.Background(), testNamespace, metav1.CreateOptions{})
	assert.NoError(t, err)

	tests := []struct {
		name       string
		namespace  string
		wantExists bool
		wantError  bool
	}{
		{
			name:       "existing namespace",
			namespace:  "test-namespace",
			wantExists: true,
			wantError:  false,
		},
		{
			name:       "non-existing namespace",
			namespace:  "non-existing-namespace",
			wantExists: false,
			wantError:  false, // Error is captured in result, not returned
		},
		{
			name:       "empty namespace",
			namespace:  "",
			wantExists: false,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckNamespace(context.Background(), fakeClient, tt.namespace)

			assert.Equal(t, tt.wantExists, result.Exists)
			assert.Equal(t, tt.namespace, result.Namespace)

			if tt.wantExists {
				assert.NoError(t, result.Error)
				assert.Empty(t, result.AvailableNS)
			} else {
				assert.Error(t, result.Error)
				// Should have available namespaces (at least test-namespace)
				assert.NotEmpty(t, result.AvailableNS)
				assert.Contains(t, result.AvailableNS, "test-namespace")
			}
		})
	}
}

func TestLogNamespaceCheck(t *testing.T) {
	// This test mainly ensures the function doesn't panic
	// In a real scenario, you might want to capture stdout and verify the output
	result := NamespaceCheckResult{
		Exists:    true,
		Namespace: "test-namespace",
	}

	// Test with existing namespace
	LogNamespaceCheck(result, "info")

	// Test with non-existing namespace
	result.Exists = false
	result.Error = assert.AnError
	result.AvailableNS = []string{"default", "kube-system"}

	// This should not panic
	LogNamespaceCheck(result, "warn")
}
