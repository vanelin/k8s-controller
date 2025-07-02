package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/vanelin/k8s-controller.git/pkg/common/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconfigFlag string
var namespaceFlag string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Kubernetes deployments in the specified namespace",
	Run: func(cmd *cobra.Command, args []string) {
		// Get kubeconfig path with proper priority using existing config logic
		kubeconfigPath := getKubeconfigPath()

		log.Info().Str("kubeconfig", kubeconfigPath).Str("namespace", namespaceFlag).Msg("Using kubeconfig path and namespace")

		clientset, err := getKubeClient(kubeconfigPath)
		if err != nil {
			log.Error().Err(err).Str("kubeconfig", kubeconfigPath).Msg("Failed to create Kubernetes client")
			os.Exit(1)
		}

		// Check if namespace exists using utility function
		result := utils.CheckNamespace(context.Background(), clientset, namespaceFlag)
		if !result.Exists {
			utils.LogNamespaceCheck(result, "error")
			os.Exit(1)
		}

		log.Info().Str("namespace", result.Namespace).Msg("Namespace exists, listing deployments")

		// List Deployments
		deployments, err := clientset.AppsV1().Deployments(namespaceFlag).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Error().Err(err).Str("namespace", namespaceFlag).Msg("Failed to list deployments")
			os.Exit(1)
		}
		fmt.Printf("Found %d deployments in '%s' namespace:\n", len(deployments.Items), namespaceFlag)
		for _, d := range deployments.Items {
			fmt.Println("-", d.Name)
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

func getKubeClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVar(&kubeconfigFlag, "kubeconfig", "", "Path to the kubeconfig file (overrides env vars and config)")
	listCmd.Flags().StringVarP(&namespaceFlag, "namespace", "n", "default", "Namespace to list deployments from")
}
