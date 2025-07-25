package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	zerologr "github.com/go-logr/zerologr"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
	"github.com/vanelin/k8s-controller/pkg/common/utils"
	"github.com/vanelin/k8s-controller/pkg/ctrl"
	"github.com/vanelin/k8s-controller/pkg/handlers"
	"github.com/vanelin/k8s-controller/pkg/informer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var serverPort string
var serverKubeconfig string
var serverInCluster bool
var serverNamespace string
var serverMetricPort string
var serverEnableLeaderElection bool
var serverLeaderElectionNamespace string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a FastHTTP server with Deployment informer and controller-runtime",
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
		if serverMetricPort != "" {
			cfg.MetricPort = serverMetricPort
		}
		// Handle leader election flag - CLI flag takes precedence over config
		if cmd.Flags().Changed("enable-leader-election") {
			cfg.EnableLeaderElection = serverEnableLeaderElection
		}
		// Handle leader election namespace flag - CLI flag takes precedence over config
		if serverLeaderElectionNamespace != "" {
			cfg.LeaderElectionNamespace = serverLeaderElectionNamespace
		}

		// Parse namespaces to watch from --namespace (comma-separated)
		namespacesToWatch := []string{"default"}
		if serverNamespace != "" {
			// Parse CLI flag (comma-separated)
			namespacesToWatch = strings.Split(serverNamespace, ",")
			for i, ns := range namespacesToWatch {
				namespacesToWatch[i] = strings.TrimSpace(ns)
			}
			// Update cfg.Namespace for display
			cfg.Namespace = serverNamespace
		} else if appConfig.Namespace != "" {
			// Parse environment variable (comma-separated)
			namespacesToWatch = strings.Split(appConfig.Namespace, ",")
			for i, ns := range namespacesToWatch {
				namespacesToWatch[i] = strings.TrimSpace(ns)
			}
			// Update cfg.Namespace for display
			cfg.Namespace = appConfig.Namespace
		}

		// Print updated configuration
		cfg.PrintConfig()

		// Print additional controller-specific configuration
		log.Info().
			Str("metrics_port", cfg.MetricPort).
			Bool("leader_election", cfg.EnableLeaderElection).
			Msg("Controller configuration")

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

		var informerManager *informer.DeploymentInformerManager
		var handlerManager *handlers.HandlerManager

		if kubeconfig != "" || inCluster {
			// Create Kubernetes client
			clientset, err := getServerKubeClient(kubeconfig, inCluster)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create Kubernetes client")
				os.Exit(1)
			}

			// Create informer manager
			informerManager = informer.NewDeploymentInformerManager(clientset)

			// Start informers for each namespace
			for _, namespace := range namespacesToWatch {
				// Check if namespace exists before starting informer
				result := utils.CheckNamespace(context.Background(), clientset, namespace)
				if !result.Exists {
					log.Warn().Err(result.Error).Str("namespace", namespace).Msg("Namespace does not exist, skipping")
					continue
				}

				log.Info().Str("namespace", namespace).Msg("Starting informer for namespace")
				informerManager.StartInformer(ctx, namespace)
			}

			// Create handler manager
			handlerManager = handlers.NewHandlerManager(informerManager, appVersion)

			log.Info().Strs("namespaces", namespacesToWatch).Msg("Started informers for namespaces")

			// Start controller-runtime manager and controller
			metricPort := cfg.MetricPort
			if metricPort == "" {
				metricPort = "8081" // fallback default
			}
			// Use zerologr for controller-runtime
			ctrlLogger := zerologr.New(&log.Logger)
			ctrlruntime.SetLogger(ctrlLogger)

			// Configure manager options with leader election
			managerOpts := manager.Options{
				Logger: ctrlLogger,
				Metrics: metricsserver.Options{
					BindAddress: ":" + metricPort,
				},
			}

			// Configure leader election if enabled
			if cfg.EnableLeaderElection {
				// Use configured leader election namespace
				leaderElectionNamespace := cfg.LeaderElectionNamespace
				if leaderElectionNamespace == "" {
					leaderElectionNamespace = "default"
				}

				managerOpts.LeaderElection = true
				managerOpts.LeaderElectionNamespace = leaderElectionNamespace
				managerOpts.LeaderElectionID = "k8s-controller-leader-election"
				managerOpts.LeaderElectionResourceLock = "leases"

				log.Info().
					Str("namespace", leaderElectionNamespace).
					Str("resource_lock", "leases").
					Strs("watched_namespaces", namespacesToWatch).
					Msg("Leader election enabled")
			} else {
				log.Info().Msg("Leader election disabled")
			}

			mgr, err := ctrlruntime.NewManager(ctrlruntime.GetConfigOrDie(), managerOpts)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create controller-runtime manager")
				os.Exit(1)
			}
			if err := ctrl.AddDeploymentControllerWithNameAndNamespaces(mgr, "deployment", namespacesToWatch); err != nil {
				log.Error().Err(err).Msg("Failed to add deployment controller")
				os.Exit(1)
			}
			go func() {
				log.Info().Str("metrics_port", metricPort).Msg("Starting controller-runtime manager...")
				if err := mgr.Start(cmd.Context()); err != nil {
					log.Error().Err(err).Msg("Manager exited with error")
					cancel() // Signal other goroutines to stop
				}
			}()
		} else {
			log.Info().Msg("Skipping Deployment informer - no Kubernetes configuration provided")
			// Create empty informer manager for handlers
			informerManager = informer.NewDeploymentInformerManager(nil)
			handlerManager = handlers.NewHandlerManager(informerManager, appVersion)
		}

		// Determine port with proper formatting - add colon for FastHTTP
		port := cfg.Port
		if port != "" {
			port = ":" + port
		}

		// Create HTTP server with graceful shutdown
		server := &fasthttp.Server{
			Handler: handlerManager.CreateHandler(),
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

		// Cancel context to stop informers
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

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVarP(&serverPort, "port", "p", "", "Port to run the server on (overrides env vars and config, default: 8080)")
	serverCmd.Flags().StringVar(&serverKubeconfig, "kubeconfig", "", "Path to the kubeconfig file (default: ~/.kube/config)")
	serverCmd.Flags().BoolVar(&serverInCluster, "in-cluster", false, "Use in-cluster Kubernetes config (default: false)")
	serverCmd.Flags().StringVarP(&serverNamespace, "namespace", "n", "", "Namespace(s) to watch for Deployments (comma-separated, default: default)")
	serverCmd.Flags().StringVar(&serverMetricPort, "metric-port", "", "Port to run the controller-runtime metrics server on (overrides env vars and config, default: 8081)")
	serverCmd.Flags().BoolVar(&serverEnableLeaderElection, "enable-leader-election", true, "Enable leader election for controller manager")
	serverCmd.Flags().StringVar(&serverLeaderElectionNamespace, "leader-election-namespace", "", "Namespace for leader election (overrides env vars and config, default: default)")
}
