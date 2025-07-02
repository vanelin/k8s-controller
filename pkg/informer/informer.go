package informer

import (
	"context"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// StartDeploymentInformer starts a shared informer for Deployments in the specified namespace.
func StartDeploymentInformer(ctx context.Context, clientset *kubernetes.Clientset, namespace string) {
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

func getDeploymentName(obj any) string {
	if d, ok := obj.(metav1.Object); ok {
		return d.GetName()
	}
	return "unknown"
}
