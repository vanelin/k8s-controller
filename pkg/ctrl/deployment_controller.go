package ctrl

import (
	"context"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// DeploymentReconciler reconciles Deployment objects
type DeploymentReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Namespaces []string // List of namespaces to watch
}

// Reconcile handles reconciliation of Deployment resources
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Check if namespace is in the watched list
	if !r.isNamespaceWatched(req.Namespace) {
		return ctrl.Result{}, nil
	}

	logger := log.With().
		Str("namespace", req.Namespace).
		Str("name", req.Name).
		Logger()

	logger.Info().Msg("Reconciling Deployment")

	// Get the Deployment
	var deployment appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &deployment); err != nil {
		// Handle the case where the Deployment is not found
		if client.IgnoreNotFound(err) != nil {
			logger.Error().Err(err).Msg("Failed to get Deployment")
			return ctrl.Result{}, err
		}
		logger.Info().Msg("Deployment not found, likely deleted")
		return ctrl.Result{}, nil
	}

	// Log deployment details
	logger.Info().
		Int32("replicas", *deployment.Spec.Replicas).
		Str("image", deployment.Spec.Template.Spec.Containers[0].Image).
		Msg("Deployment reconciled successfully")

	return ctrl.Result{}, nil
}

// isNamespaceWatched checks if namespace is being watched
func (r *DeploymentReconciler) isNamespaceWatched(namespace string) bool {
	for _, ns := range r.Namespaces {
		if ns == namespace {
			return true
		}
	}
	return false
}

// AddDeploymentController adds the Deployment controller to the manager
func AddDeploymentController(mgr manager.Manager) error {
	return AddDeploymentControllerWithName(mgr, "deployment")
}

// AddDeploymentControllerWithName adds the Deployment controller to the manager with a custom name
func AddDeploymentControllerWithName(mgr manager.Manager, name string) error {
	return AddDeploymentControllerWithNameAndNamespaces(mgr, name, []string{"default"})
}

// AddDeploymentControllerWithNameAndNamespaces adds the Deployment controller to the manager with custom name and namespaces
func AddDeploymentControllerWithNameAndNamespaces(mgr manager.Manager, name string, namespaces []string) error {
	r := &DeploymentReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Namespaces: namespaces,
	}

	// Create predicate for filtering by namespaces
	namespacePredicate := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return r.isNamespaceWatched(obj.GetNamespace())
	})

	log.Info().
		Str("controller_name", name).
		Strs("namespaces", namespaces).
		Msg("Adding Deployment controller with namespace filter")

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&appsv1.Deployment{}).
		WithEventFilter(namespacePredicate).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
