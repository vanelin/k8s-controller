package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
	"github.com/vanelin/k8s-controller.git/pkg/common/utils"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var serverPort string
var serverKubeconfig string
var serverInCluster bool
var serverNamespace string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a FastHTTP server with Deployment informer",
	Run: func(cmd *cobra.Command, args []string) {
		// Use the already loaded configuration from root.go
		cfg := appConfig

		// Override with CLI flags
		if serverPort != "" {
			cfg.Port = serverPort
		}
		if logLevel != "" {
			cfg.LoggingLevel = logLevel
		}

		// Print updated configuration
		cfg.PrintConfig()

		// Start Deployment informer if Kubernetes flags are provided
		// Priority: CLI flags > env vars > .env file > defaults
		kubeconfig := serverKubeconfig
		if kubeconfig == "" {
			kubeconfig = appConfig.KUBECONFIG
		}
		// Expand tilde in kubeconfig path
		kubeconfig = utils.ExpandTilde(kubeconfig)

		inCluster := serverInCluster
		if !inCluster {
			inCluster = appConfig.InCluster
		}

		namespace := serverNamespace
		if namespace == "" {
			namespace = appConfig.Namespace
		}

		if kubeconfig != "" || inCluster {
			// Create Kubernetes client
			clientset, err := getServerKubeClient(kubeconfig, inCluster)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create Kubernetes client")
				os.Exit(1)
			}

			// Check if namespace exists before starting informer
			originalNamespace := namespace
			result := utils.CheckNamespace(context.Background(), clientset, namespace)
			if !result.Exists {
				log.Warn().Err(result.Error).Str("namespace", namespace).Msg("Namespace does not exist, using default namespace")
				namespace = "default"
				log.Info().Str("original_namespace", originalNamespace).Str("fallback_namespace", namespace).Msg("Switched to default namespace")
			}

			log.Info().Str("namespace", namespace).Msg("Starting Deployment informer")

			// Start Deployment informer in background
			ctx := context.Background()
			go startDeploymentInformer(ctx, clientset, namespace)
		} else {
			log.Info().Msg("Skipping Deployment informer - no Kubernetes configuration provided")
		}

		// Determine port with proper formatting - add colon for FastHTTP
		port := cfg.Port
		if port != "" {
			port = ":" + port
		}

		handler := func(ctx *fasthttp.RequestCtx) {
			if _, err := fmt.Fprintf(ctx, "Hello from FastHTTP!"); err != nil {
				log.Error().Err(err).Msg("Failed to write response")
			}
		}
		log.Info().Msgf("Starting FastHTTP server on %s (version: %s)", port, appVersion)
		if err := fasthttp.ListenAndServe(port, handler); err != nil {
			log.Error().Err(err).Msg("Error starting FastHTTP server")
			os.Exit(1)
		}
	},
}

func getServerKubeClient(kubeconfigPath string, inCluster bool) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if inCluster {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// startDeploymentInformer starts a shared informer for Deployments in the specified namespace
func startDeploymentInformer(ctx context.Context, clientset *kubernetes.Clientset, namespace string) {
	// Create informer for Deployments in the specified namespace
	deploymentInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return clientset.AppsV1().Deployments(namespace).List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return clientset.AppsV1().Deployments(namespace).Watch(ctx, options)
			},
		},
		&appsv1.Deployment{},
		0, // resync period
		cache.Indexers{},
	)

	// Add event handlers for add, update, and delete events
	if _, err := deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deployment := obj.(*appsv1.Deployment)
			log.Info().
				Str("event", "add").
				Str("name", deployment.Name).
				Str("namespace", deployment.Namespace).
				Int32("replicas", *deployment.Spec.Replicas).
				Msg("Deployment added")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDeployment := oldObj.(*appsv1.Deployment)
			newDeployment := newObj.(*appsv1.Deployment)
			log.Info().
				Str("event", "update").
				Str("name", newDeployment.Name).
				Str("namespace", newDeployment.Namespace).
				Int32("old_replicas", *oldDeployment.Spec.Replicas).
				Int32("new_replicas", *newDeployment.Spec.Replicas).
				Msg("Deployment updated")
		},
		DeleteFunc: func(obj interface{}) {
			deployment := obj.(*appsv1.Deployment)
			log.Info().
				Str("event", "delete").
				Str("name", deployment.Name).
				Str("namespace", deployment.Namespace).
				Msg("Deployment deleted")
		},
	}); err != nil {
		log.Error().Err(err).Msg("Failed to add event handler")
		return
	}

	// Start the informer
	log.Info().Str("namespace", namespace).Msg("Starting Deployment informer")
	go deploymentInformer.Run(ctx.Done())

	// Wait for the informer to sync
	if !cache.WaitForCacheSync(ctx.Done(), deploymentInformer.HasSynced) {
		log.Error().Str("namespace", namespace).Msg("Failed to sync Deployment informer")
		return
	}

	log.Info().Str("namespace", namespace).Msg("Deployment informer started successfully")
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVarP(&serverPort, "port", "p", "", "Port to run the server on (overrides env vars and config, default: 8080)")
	serverCmd.Flags().StringVar(&serverKubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
	serverCmd.Flags().BoolVar(&serverInCluster, "in-cluster", false, "Use in-cluster Kubernetes config")
	serverCmd.Flags().StringVarP(&serverNamespace, "namespace", "n", "", "Namespace to watch for Deployments (default: default)")
}
