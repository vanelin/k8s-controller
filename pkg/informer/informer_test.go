package informer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	testutil "github.com/vanelin/k8s-controller.git/pkg/testutil"
)

func TestStartDeploymentInformer(t *testing.T) {
	_, clientset, cleanup := testutil.SetupEnv(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	added := make(chan string, 2)
	updated := make(chan int32, 1)
	deleted := make(chan string, 1)

	factory := informers.NewSharedInformerFactoryWithOptions(
		clientset,
		30*time.Second,
		informers.WithNamespace("default"),
	)
	informer := factory.Apps().V1().Deployments().Informer()
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if d, ok := obj.(*appsv1.Deployment); ok {
				added <- d.Name
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldD := oldObj.(*appsv1.Deployment)
			newD := newObj.(*appsv1.Deployment)
			if *oldD.Spec.Replicas != *newD.Spec.Replicas {
				updated <- *newD.Spec.Replicas
			}
		},
		DeleteFunc: func(obj interface{}) {
			if d, ok := obj.(*appsv1.Deployment); ok {
				deleted <- d.Name
			}
		},
	})
	if err != nil {
		t.Fatalf("Failed to add event handlers to informer: %v", err)
	}

	go func() {
		defer wg.Done()
		factory.Start(ctx.Done())
		factory.WaitForCacheSync(ctx.Done())
		<-ctx.Done()
	}()

	// Wait for add events
	found := map[string]bool{}
	for range 2 {
		select {
		case name := <-added:
			found[name] = true
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for deployment add events")
		}
	}
	require.True(t, found["sample-deployment-1"])
	require.True(t, found["sample-deployment-2"])

	// Simulate update event (change replicas)
	dep, err := clientset.AppsV1().Deployments("default").Get(ctx, "sample-deployment-1", metav1.GetOptions{})
	require.NoError(t, err)
	newReplicas := int32(3)
	dep.Spec.Replicas = &newReplicas
	_, err = clientset.AppsV1().Deployments("default").Update(ctx, dep, metav1.UpdateOptions{})
	require.NoError(t, err)

	select {
	case r := <-updated:
		require.Equal(t, int32(3), r)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for deployment update event")
	}

	// Simulate delete event
	err = clientset.AppsV1().Deployments("default").Delete(ctx, "sample-deployment-2", metav1.DeleteOptions{})
	require.NoError(t, err)
	select {
	case name := <-deleted:
		require.Equal(t, "sample-deployment-2", name)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for deployment delete event")
	}

	cancel()
	wg.Wait()
}

func TestGetDeploymentName(t *testing.T) {
	dep := &metav1.PartialObjectMetadata{}
	dep.SetName("my-deployment")
	name := getDeploymentName(dep)
	if name != "my-deployment" {
		t.Errorf("expected 'my-deployment', got %q", name)
	}
	name = getDeploymentName("not-an-object")
	if name != "unknown" {
		t.Errorf("expected 'unknown', got %q", name)
	}
}

func TestStartDeploymentInformer_CoversFunction(t *testing.T) {
	_, clientset, cleanup := testutil.SetupEnv(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run StartDeploymentInformer in a goroutine
	go func() {
		StartDeploymentInformer(ctx, clientset, "default")
	}()

	// Give the informer some time to start and process events
	time.Sleep(1 * time.Second)
	cancel()
}
