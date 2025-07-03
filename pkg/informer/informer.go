package informer

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// DeploymentInformerManager manages multiple deployment informers for different namespaces
type DeploymentInformerManager struct {
	mu        sync.RWMutex
	informers map[string]cache.SharedIndexInformer
	clientset *kubernetes.Clientset
}

// NewDeploymentInformerManager creates a new informer manager
func NewDeploymentInformerManager(clientset *kubernetes.Clientset) *DeploymentInformerManager {
	return &DeploymentInformerManager{
		informers: make(map[string]cache.SharedIndexInformer),
		clientset: clientset,
	}
}

// StartDeploymentInformer starts a shared informer for Deployments in the specified namespace.
// This function is kept for backward compatibility.
func StartDeploymentInformer(ctx context.Context, clientset *kubernetes.Clientset, namespace string) {
	manager := NewDeploymentInformerManager(clientset)
	manager.StartInformer(ctx, namespace)

	// Wait for context cancellation
	<-ctx.Done()
	log.Info().Msg("Deployment informer shutting down")
}

// StartInformer starts an informer for a specific namespace
func (m *DeploymentInformerManager) StartInformer(ctx context.Context, namespace string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if informer already exists for this namespace
	if _, exists := m.informers[namespace]; exists {
		log.Info().Str("namespace", namespace).Msg("Deployment informer already exists for namespace")
		return
	}

	log.Info().Str("namespace", namespace).Msg("Starting Deployment informer")

	// Create informer factory
	informerFactory := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return m.clientset.AppsV1().Deployments(namespace).List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return m.clientset.AppsV1().Deployments(namespace).Watch(ctx, options)
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

	// Store the informer
	m.informers[namespace] = informerFactory

	// Start the informer
	go informerFactory.Run(ctx.Done())

	// Wait for the informer to sync
	if !cache.WaitForCacheSync(ctx.Done(), informerFactory.HasSynced) {
		log.Error().Msg("Failed to sync informer cache")
		return
	}

	log.Info().Msg("Deployment informer started successfully")
}

// GetDeploymentNames returns a slice of deployment names from the informer's cache for a specific namespace.
func (m *DeploymentInformerManager) GetDeploymentNames(namespace string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	informer, exists := m.informers[namespace]
	if !exists {
		return []string{}
	}

	var names []string
	for _, obj := range informer.GetStore().List() {
		if d, ok := obj.(*appsv1.Deployment); ok {
			names = append(names, d.Name)
		}
	}
	return names
}

// GetDeploymentNamesFromDefault returns deployment names from the default namespace informer.
// This function is kept for backward compatibility.
func GetDeploymentNames() []string {
	// This is a fallback for the old API - returns empty slice if no manager exists
	return []string{}
}

// GetAvailableNamespaces returns a list of namespaces that have active informers
func (m *DeploymentInformerManager) GetAvailableNamespaces() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	namespaces := make([]string, 0, len(m.informers))
	for namespace := range m.informers {
		namespaces = append(namespaces, namespace)
	}
	return namespaces
}

// HasInformer checks if an informer exists for the given namespace
func (m *DeploymentInformerManager) HasInformer(namespace string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.informers[namespace]
	return exists
}

func getDeploymentName(obj any) string {
	if d, ok := obj.(metav1.Object); ok {
		return d.GetName()
	}
	return "unknown"
}
