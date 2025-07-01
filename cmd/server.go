package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

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
		// Create context with cancel for graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create WaitGroup to track goroutines
		var wg sync.WaitGroup

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

			// Start Deployment informer in background with WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				startDeploymentInformer(ctx, clientset, namespace)
			}()
		} else {
			log.Info().Msg("Skipping Deployment informer - no Kubernetes configuration provided")
		}

		// Determine port with proper formatting - add colon for FastHTTP
		port := cfg.Port
		if port != "" {
			port = ":" + port
		}

		// Create HTTP server with graceful shutdown
		server := &fasthttp.Server{
			Handler: func(ctx *fasthttp.RequestCtx) {
				if _, err := fmt.Fprintf(ctx, "Hello from FastHTTP!"); err != nil {
					log.Error().Err(err).Msg("Failed to write response")
				}
			},
		}

		// Start HTTP server in background
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info().Msgf("Starting FastHTTP server on %s (version: %s)", port, appVersion)
			if err := server.ListenAndServe(port); err != nil {
				log.Error().Err(err).Msg("Error starting FastHTTP server")
				cancel() // Signal other goroutines to stop
			}
		}()

		// Setup signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

		// Wait for shutdown signal
		select {
		case sig := <-sigChan:
			log.Info().Str("signal", sig.String()).Msg("Received shutdown signal, starting graceful shutdown")
		case <-ctx.Done():
			log.Info().Msg("Context cancelled, starting graceful shutdown")
		}

		// Graceful shutdown
		log.Info().Msg("Shutting down HTTP server...")
		if err := server.Shutdown(); err != nil {
			log.Error().Err(err).Msg("Error shutting down HTTP server")
		}

		// Cancel context to stop informer
		cancel()

		// Wait for all goroutines to finish
		log.Info().Msg("Waiting for goroutines to finish...")
		wg.Wait()

		log.Info().Msg("Graceful shutdown completed")
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
	log.Info().Str("namespace", namespace).Msg("Starting Deployment informer")

	// Create informer factory
	informerFactory := cache.NewSharedIndexInformer(
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

	// Add event handlers
	_, err := informerFactory.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deployment := obj.(*appsv1.Deployment)
			log.Info().
				Str("event", "ADDED").
				Str("namespace", deployment.Namespace).
				Str("name", deployment.Name).
				Int32("replicas", *deployment.Spec.Replicas).
				Msg("Deployment added")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDeployment := oldObj.(*appsv1.Deployment)
			newDeployment := newObj.(*appsv1.Deployment)

			// Determine what type of change occurred
			var changeType string
			if *oldDeployment.Spec.Replicas != *newDeployment.Spec.Replicas {
				changeType = "spec_replicas"
			} else if oldDeployment.Status.Replicas != newDeployment.Status.Replicas {
				changeType = "status_replicas"
			} else if oldDeployment.Status.ReadyReplicas != newDeployment.Status.ReadyReplicas {
				changeType = "ready_replicas"
			} else if oldDeployment.Status.AvailableReplicas != newDeployment.Status.AvailableReplicas {
				changeType = "available_replicas"
			} else {
				changeType = "status_only"
			}

			log.Info().
				Str("event", "MODIFIED").
				Str("namespace", newDeployment.Namespace).
				Str("name", newDeployment.Name).
				Int32("replicas", *newDeployment.Spec.Replicas).
				Str("change", changeType).
				Msg("Deployment updated")
		},
		DeleteFunc: func(obj interface{}) {
			deployment := obj.(*appsv1.Deployment)
			log.Info().
				Str("event", "DELETED").
				Str("namespace", deployment.Namespace).
				Str("name", deployment.Name).
				Msg("Deployment deleted")
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to add event handlers to informer")
		return
	}

	// Start the informer
	go informerFactory.Run(ctx.Done())

	// Wait for the informer to sync
	if !cache.WaitForCacheSync(ctx.Done(), informerFactory.HasSynced) {
		log.Error().Msg("Failed to sync informer cache")
		return
	}

	log.Info().Msg("Deployment informer started successfully")

	// Wait for context cancellation
	<-ctx.Done()
	log.Info().Msg("Deployment informer shutting down")
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVarP(&serverPort, "port", "p", "", "Port to run the server on (overrides env vars and config, default: 8080)")
	serverCmd.Flags().StringVar(&serverKubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
	serverCmd.Flags().BoolVar(&serverInCluster, "in-cluster", false, "Use in-cluster Kubernetes config")
	serverCmd.Flags().StringVarP(&serverNamespace, "namespace", "n", "", "Namespace to watch for Deployments (default: default)")
}
