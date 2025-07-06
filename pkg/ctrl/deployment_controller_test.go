package ctrl

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testutil "github.com/vanelin/k8s-controller/pkg/testutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDeploymentReconciler_BasicFlow(t *testing.T) {
	mgr, k8sClient, _, cleanup := testutil.StartTestManager(t)
	defer cleanup()

	// Register the controller before starting the manager
	err := AddDeploymentControllerWithName(mgr, "deployment-basic")
	require.NoError(t, err)

	// Create a context with cancellation for proper cleanup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = mgr.Start(ctx)
	}()

	ns := "default"
	testCtx := context.Background()
	name := "test-deployment"

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}}},
			},
		},
	}
	if err := k8sClient.Create(testCtx, dep); err != nil {
		t.Fatalf("Failed to create Deployment: %v", err)
	}

	// Wait a bit to allow reconcile to be triggered
	time.Sleep(1 * time.Second)

	// Just check the Deployment still exists (reconcile didn't error or delete it)
	var got appsv1.Deployment
	err = k8sClient.Get(testCtx, client.ObjectKey{Name: name, Namespace: ns}, &got)
	require.NoError(t, err)
}

func TestDeploymentReconciler_MultipleDeployments(t *testing.T) {
	mgr, k8sClient, _, cleanup := testutil.StartTestManager(t)
	defer cleanup()

	// Register the controller before starting the manager
	err := AddDeploymentControllerWithName(mgr, "deployment-multiple")
	require.NoError(t, err)

	// Create a context with cancellation for proper cleanup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = mgr.Start(ctx)
	}()

	ns := "default"
	testCtx := context.Background()

	// Create multiple deployments
	deployments := []string{"deployment-1", "deployment-2", "deployment-3"}

	for _, name := range deployments {
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(2),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": name},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx:1.21"}}},
				},
			},
		}
		if err := k8sClient.Create(testCtx, dep); err != nil {
			t.Fatalf("Failed to create Deployment %s: %v", name, err)
		}
	}

	// Wait a bit to allow reconcile to be triggered
	time.Sleep(2 * time.Second)

	// Check all deployments still exist
	for _, name := range deployments {
		var got appsv1.Deployment
		err = k8sClient.Get(testCtx, client.ObjectKey{Name: name, Namespace: ns}, &got)
		require.NoError(t, err)
		require.Equal(t, name, got.Name)
		require.Equal(t, int32(2), *got.Spec.Replicas)
	}
}

func TestDeploymentReconciler_UpdateDeployment(t *testing.T) {
	mgr, k8sClient, _, cleanup := testutil.StartTestManager(t)
	defer cleanup()

	// Register the controller before starting the manager
	err := AddDeploymentControllerWithName(mgr, "deployment-update")
	require.NoError(t, err)

	// Create a context with cancellation for proper cleanup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = mgr.Start(ctx)
	}()

	ns := "default"
	testCtx := context.Background()
	name := "update-test-deployment"

	// Create initial deployment
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx:1.20"}}},
			},
		},
	}
	if err := k8sClient.Create(testCtx, dep); err != nil {
		t.Fatalf("Failed to create Deployment: %v", err)
	}

	// Wait for initial reconcile
	time.Sleep(1 * time.Second)

	// Update the deployment
	var got appsv1.Deployment
	err = k8sClient.Get(testCtx, client.ObjectKey{Name: name, Namespace: ns}, &got)
	require.NoError(t, err)

	// Update replicas and image
	newReplicas := int32(3)
	got.Spec.Replicas = &newReplicas
	got.Spec.Template.Spec.Containers[0].Image = "nginx:1.21"

	if err := k8sClient.Update(testCtx, &got); err != nil {
		t.Fatalf("Failed to update Deployment: %v", err)
	}

	// Wait for reconcile after update
	time.Sleep(1 * time.Second)

	// Verify the update was processed
	var updated appsv1.Deployment
	err = k8sClient.Get(testCtx, client.ObjectKey{Name: name, Namespace: ns}, &updated)
	require.NoError(t, err)
	require.Equal(t, int32(3), *updated.Spec.Replicas)
	require.Equal(t, "nginx:1.21", updated.Spec.Template.Spec.Containers[0].Image)
}

func TestAddDeploymentController(t *testing.T) {
	mgr, _, _, cleanup := testutil.StartTestManager(t)
	defer cleanup()

	// Test that we can add the controller without error
	err := AddDeploymentControllerWithName(mgr, "deployment-test")
	require.NoError(t, err)
}

func int32Ptr(i int32) *int32 { return &i }
