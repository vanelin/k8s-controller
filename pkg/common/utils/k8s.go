package utils

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NamespaceCheckResult represents the result of namespace validation
type NamespaceCheckResult struct {
	Exists      bool
	Namespace   string
	Error       error
	AvailableNS []string
}

// CheckNamespace validates if a namespace exists and returns available namespaces if it doesn't
func CheckNamespace(ctx context.Context, clientset kubernetes.Interface, namespace string) NamespaceCheckResult {
	result := NamespaceCheckResult{
		Namespace: namespace,
	}

	// Check if namespace exists
	_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		result.Exists = false
		result.Error = err

		// List available namespaces
		namespaces, listErr := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if listErr != nil {
			result.Error = fmt.Errorf("failed to check namespace '%s': %w, failed to list namespaces: %w", namespace, err, listErr)
			return result
		}

		// Extract namespace names
		for _, ns := range namespaces.Items {
			result.AvailableNS = append(result.AvailableNS, ns.Name)
		}

		return result
	}

	result.Exists = true
	return result
}

// LogNamespaceCheck logs the result of namespace validation with appropriate log level
func LogNamespaceCheck(result NamespaceCheckResult, logLevel string) {
	if result.Exists {
		log.Info().Str("namespace", result.Namespace).Msg("Namespace exists")
		return
	}

	// Log error with specified level
	switch logLevel {
	case "error":
		log.Error().Err(result.Error).Str("namespace", result.Namespace).Msg("Namespace does not exist")
	case "warn":
		log.Warn().Err(result.Error).Str("namespace", result.Namespace).Msg("Namespace does not exist")
	default:
		log.Warn().Err(result.Error).Str("namespace", result.Namespace).Msg("Namespace does not exist")
	}

	// Print available namespaces
	fmt.Printf("Namespace '%s' does not exist.\n", result.Namespace)
	if len(result.AvailableNS) > 0 {
		fmt.Printf("Available namespaces:\n")
		for _, ns := range result.AvailableNS {
			fmt.Printf("- %s\n", ns)
		}
	}
}
