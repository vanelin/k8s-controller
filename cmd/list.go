package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/vanelin/k8s-controller/pkg/common/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconfigFlag string
var namespaceFlag string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Kubernetes deployments in the specified namespace(s)",
	Run: func(cmd *cobra.Command, args []string) {
		// Get kubeconfig path with proper priority using existing config logic
		kubeconfigPath := getKubeconfigPath()

		// Get namespace with proper priority: CLI flag > env vars > .env file > defaults
		namespaceToUse := getNamespaceWithPriority()

		log.Info().Str("kubeconfig", kubeconfigPath).Str("namespace", namespaceToUse).Msg("Using kubeconfig path and namespace")

		clientset, err := getKubeClient(kubeconfigPath)
		if err != nil {
			log.Error().Err(err).Str("kubeconfig", kubeconfigPath).Msg("Failed to create Kubernetes client")
			os.Exit(1)
		}

		// Parse namespaces from the determined namespace value (comma-separated)
		namespaces := parseNamespaces(namespaceToUse)

		// List deployments for each namespace
		totalDeployments := 0
		for _, namespace := range namespaces {
			// Check if namespace exists using utility function
			result := utils.CheckNamespace(context.Background(), clientset, namespace)
			if !result.Exists {
				log.Warn().Err(result.Error).Str("namespace", namespace).Msg("Namespace does not exist, skipping")
				continue
			}

			log.Info().Str("namespace", namespace).Msg("Listing deployments in namespace")

			// List Deployments
			deployments, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				log.Error().Err(err).Str("namespace", namespace).Msg("Failed to list deployments")
				continue
			}

			fmt.Printf("Found %d deployments in '%s' namespace:\n", len(deployments.Items), namespace)
			for _, d := range deployments.Items {
				fmt.Println("-", d.Name)
			}
			totalDeployments += len(deployments.Items)
		}

		if len(namespaces) > 1 {
			fmt.Printf("\nTotal deployments across all namespaces: %d\n", totalDeployments)
		}
	},
}

// Uses the existing appConfig that was loaded by root.go with Viper
func getKubeconfigPath() string {
	// 1. CLI flag takes highest priority
	if kubeconfigFlag != "" {
		return utils.ExpandTilde(kubeconfigFlag)
	}

	// 2-4. Use the existing appConfig which already has the proper priority logic
	// from Viper (env vars -> .env file -> defaults)
	return utils.ExpandTilde(appConfig.KUBECONFIG)
}

// getNamespaceWithPriority returns the namespace with proper priority: CLI flag > env vars > .env file > defaults
func getNamespaceWithPriority() string {
	// 1. CLI flag takes highest priority
	if namespaceFlag != "" {
		return namespaceFlag
	}
	// 2. Config (env/.env)
	if appConfig.Namespace != "" {
		return appConfig.Namespace
	}
	// 3. Default fallback
	return "default"
}

func getKubeClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// parseNamespaces splits a comma-separated string of namespaces and trims whitespace
func parseNamespaces(namespaceString string) []string {
	if namespaceString == "" {
		return []string{"default"}
	}

	namespaces := strings.Split(namespaceString, ",")
	for i, ns := range namespaces {
		namespaces[i] = strings.TrimSpace(ns)
	}
	return namespaces
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVar(&kubeconfigFlag, "kubeconfig", "", "Path to the kubeconfig file (overrides env vars and config)")
	listCmd.Flags().StringVarP(&namespaceFlag, "namespace", "n", "", "Namespace(s) to list deployments from (comma-separated)")
}
