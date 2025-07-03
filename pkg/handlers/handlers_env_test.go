package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/vanelin/k8s-controller.git/pkg/informer"
	testutil "github.com/vanelin/k8s-controller.git/pkg/testutil"
)

func TestHandlerManager_Integration_WithRealKubernetes(t *testing.T) {
	// Setup test environment with envtest
	_, clientset, cleanup := testutil.SetupEnv(t)
	defer cleanup()

	// Create test namespaces
	testNamespaces := []string{"test-ns-1", "test-ns-2"}
	for _, ns := range testNamespaces {
		_, err := clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	// Create test deployments in different namespaces
	testDeployments := map[string][]string{
		"test-ns-1": {"deployment-1", "deployment-2"},
		"test-ns-2": {"deployment-3"},
	}

	for namespace, deployments := range testDeployments {
		for _, deploymentName := range deployments {
			_, err := clientset.AppsV1().Deployments(namespace).Create(context.Background(), &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      deploymentName,
					Namespace: namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": deploymentName,
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": deploymentName,
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "nginx",
									Image: "nginx:1.21",
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 80,
										},
									},
								},
							},
						},
					},
				},
			}, metav1.CreateOptions{})
			require.NoError(t, err)
		}
	}

	// Create informer manager and start informers for test namespaces
	informerManager := informer.NewDeploymentInformerManager(clientset)

	// Start informers for test namespaces
	for _, namespace := range testNamespaces {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		informerManager.StartInformer(ctx, namespace)
	}

	// Wait for informers to sync
	time.Sleep(2 * time.Second)

	// Create handler manager
	handlerManager := NewHandlerManager(informerManager, "test-version-1.0.0")

	// Test cases
	t.Run("RootEndpoint", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 200, ctx.Response.StatusCode())
		assert.Contains(t, string(ctx.Response.Header.ContentType()), "application/json")

		var response map[string]interface{}
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Kubernetes Controller API", response["message"])
		assert.Equal(t, "test-version-1.0.0", response["version"])
	})

	t.Run("NamespacesEndpoint", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/namespaces")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 200, ctx.Response.StatusCode())

		var response NamespaceResponse
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		assert.Equal(t, 2, response.Count)
		assert.Contains(t, response.Namespaces, "test-ns-1")
		assert.Contains(t, response.Namespaces, "test-ns-2")
	})

	t.Run("DeploymentsInNamespace1", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/deployments/test-ns-1")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 200, ctx.Response.StatusCode())

		var response DeploymentResponse
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		assert.Equal(t, "test-ns-1", response.Namespace)
		assert.Equal(t, 2, response.Count)
		assert.Contains(t, response.Deployments, "deployment-1")
		assert.Contains(t, response.Deployments, "deployment-2")
	})

	t.Run("DeploymentsInNamespace2", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/deployments/test-ns-2")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 200, ctx.Response.StatusCode())

		var response DeploymentResponse
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		assert.Equal(t, "test-ns-2", response.Namespace)
		assert.Equal(t, 1, response.Count)
		assert.Contains(t, response.Deployments, "deployment-3")
	})

	t.Run("DeploymentsDefaultNamespace", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/deployments")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 200, ctx.Response.StatusCode())

		var response DeploymentsAllResponse
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		// Should return deployments from all watched namespaces (test-ns-1 and test-ns-2)
		assert.Equal(t, 3, response.TotalCount)
		assert.Equal(t, 2, len(response.Namespaces))

		// Check test-ns-1
		var ns1Resp *DeploymentResponse
		for _, ns := range response.Namespaces {
			if ns.Namespace == "test-ns-1" {
				ns1Resp = &ns
				break
			}
		}
		require.NotNil(t, ns1Resp)
		assert.Equal(t, 2, ns1Resp.Count)
		assert.Contains(t, ns1Resp.Deployments, "deployment-1")
		assert.Contains(t, ns1Resp.Deployments, "deployment-2")

		// Check test-ns-2
		var ns2Resp *DeploymentResponse
		for _, ns := range response.Namespaces {
			if ns.Namespace == "test-ns-2" {
				ns2Resp = &ns
				break
			}
		}
		require.NotNil(t, ns2Resp)
		assert.Equal(t, 1, ns2Resp.Count)
		assert.Contains(t, ns2Resp.Deployments, "deployment-3")
	})

	t.Run("NamespaceNotWatched", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/deployments/unknown-namespace")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 404, ctx.Response.StatusCode())

		var response ErrorResponse
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Request Error", response.Error)
		assert.Contains(t, response.Message, "Namespace not being watched")
	})

	t.Run("InvalidPath", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/deployments/invalid/path")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 400, ctx.Response.StatusCode())

		var response ErrorResponse
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Request Error", response.Error)
		assert.Contains(t, response.Message, "Invalid path format")
	})

	t.Run("NotFoundEndpoint", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/unknown-endpoint")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 404, ctx.Response.StatusCode())

		var response ErrorResponse
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Not Found", response.Error)
		assert.Equal(t, "The requested endpoint does not exist", response.Message)
	})
}

// Helper function to create int32 pointer
func int32Ptr(i int32) *int32 {
	return &i
}

func TestHandlerManager_MultipleNamespacesFromEnvironment(t *testing.T) {
	// Setup test environment with envtest
	_, clientset, cleanup := testutil.SetupEnv(t)
	defer cleanup()

	// Create test namespaces that will be specified in environment variable
	testNamespaces := []string{"env-ns-1", "env-ns-2", "env-ns-3"}
	for _, ns := range testNamespaces {
		_, err := clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	// Create test deployments in different namespaces
	testDeployments := map[string][]string{
		"env-ns-1": {"env-deployment-1", "env-deployment-2"},
		"env-ns-2": {"env-deployment-3"},
		"env-ns-3": {"env-deployment-4", "env-deployment-5", "env-deployment-6"},
	}

	for namespace, deployments := range testDeployments {
		for _, deploymentName := range deployments {
			_, err := clientset.AppsV1().Deployments(namespace).Create(context.Background(), &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      deploymentName,
					Namespace: namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": deploymentName,
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": deploymentName,
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "nginx",
									Image: "nginx:1.21",
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 80,
										},
									},
								},
							},
						},
					},
				},
			}, metav1.CreateOptions{})
			require.NoError(t, err)
		}
	}

	// Create informer manager and start informers for test namespaces
	informerManager := informer.NewDeploymentInformerManager(clientset)

	// Simulate the behavior from server.go where namespaces are parsed from environment variable
	// This mimics the logic: strings.Split(appConfig.Namespace, ",")
	namespaceString := "env-ns-1,env-ns-2,env-ns-3"
	namespacesToWatch := strings.Split(namespaceString, ",")
	for i, ns := range namespacesToWatch {
		namespacesToWatch[i] = strings.TrimSpace(ns)
	}

	// Start informers for all namespaces from environment variable
	for _, namespace := range namespacesToWatch {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		informerManager.StartInformer(ctx, namespace)
	}

	// Wait for informers to sync
	time.Sleep(2 * time.Second)

	// Create handler manager
	handlerManager := NewHandlerManager(informerManager, "test-version-1.0.0")

	t.Run("NamespacesEndpoint_ShouldReturnAllWatchedNamespaces", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/namespaces")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 200, ctx.Response.StatusCode())

		var response NamespaceResponse
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		// Should return all 3 namespaces that were specified in environment variable
		assert.Equal(t, 3, response.Count)
		assert.Contains(t, response.Namespaces, "env-ns-1")
		assert.Contains(t, response.Namespaces, "env-ns-2")
		assert.Contains(t, response.Namespaces, "env-ns-3")
	})

	t.Run("DeploymentsInEnvNs1", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/deployments/env-ns-1")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 200, ctx.Response.StatusCode())

		var response DeploymentResponse
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		assert.Equal(t, "env-ns-1", response.Namespace)
		assert.Equal(t, 2, response.Count)
		assert.Contains(t, response.Deployments, "env-deployment-1")
		assert.Contains(t, response.Deployments, "env-deployment-2")
	})

	t.Run("DeploymentsInEnvNs2", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/deployments/env-ns-2")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 200, ctx.Response.StatusCode())

		var response DeploymentResponse
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		assert.Equal(t, "env-ns-2", response.Namespace)
		assert.Equal(t, 1, response.Count)
		assert.Contains(t, response.Deployments, "env-deployment-3")
	})

	t.Run("DeploymentsInEnvNs3", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/deployments/env-ns-3")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 200, ctx.Response.StatusCode())

		var response DeploymentResponse
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		assert.Equal(t, "env-ns-3", response.Namespace)
		assert.Equal(t, 3, response.Count)
		assert.Contains(t, response.Deployments, "env-deployment-4")
		assert.Contains(t, response.Deployments, "env-deployment-5")
		assert.Contains(t, response.Deployments, "env-deployment-6")
	})

	t.Run("DefaultNamespace_ShouldReturnAllWatchedNamespaces", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/deployments")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 200, ctx.Response.StatusCode())

		var response DeploymentsAllResponse
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		// Should return deployments from all watched namespaces (env-ns-1, env-ns-2, env-ns-3)
		assert.Equal(t, 6, response.TotalCount)
		assert.Equal(t, 3, len(response.Namespaces))

		// Check env-ns-1
		var ns1Resp *DeploymentResponse
		for _, ns := range response.Namespaces {
			if ns.Namespace == "env-ns-1" {
				ns1Resp = &ns
				break
			}
		}
		require.NotNil(t, ns1Resp)
		assert.Equal(t, 2, ns1Resp.Count)
		assert.Contains(t, ns1Resp.Deployments, "env-deployment-1")
		assert.Contains(t, ns1Resp.Deployments, "env-deployment-2")

		// Check env-ns-2
		var ns2Resp *DeploymentResponse
		for _, ns := range response.Namespaces {
			if ns.Namespace == "env-ns-2" {
				ns2Resp = &ns
				break
			}
		}
		require.NotNil(t, ns2Resp)
		assert.Equal(t, 1, ns2Resp.Count)
		assert.Contains(t, ns2Resp.Deployments, "env-deployment-3")

		// Check env-ns-3
		var ns3Resp *DeploymentResponse
		for _, ns := range response.Namespaces {
			if ns.Namespace == "env-ns-3" {
				ns3Resp = &ns
				break
			}
		}
		require.NotNil(t, ns3Resp)
		assert.Equal(t, 3, ns3Resp.Count)
		assert.Contains(t, ns3Resp.Deployments, "env-deployment-4")
		assert.Contains(t, ns3Resp.Deployments, "env-deployment-5")
		assert.Contains(t, ns3Resp.Deployments, "env-deployment-6")
	})

	t.Run("NamespaceNotInEnvironmentVariable_ShouldReturn404", func(t *testing.T) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/deployments/default")
		ctx.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx)

		assert.Equal(t, 404, ctx.Response.StatusCode())

		var response ErrorResponse
		err := json.Unmarshal(ctx.Response.Body(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Request Error", response.Error)
		assert.Contains(t, response.Message, "Namespace not being watched")
	})
}

func TestHandlerManager_EnvironmentVariableParsingEdgeCases(t *testing.T) {
	// Setup test environment with envtest
	_, clientset, cleanup := testutil.SetupEnv(t)
	defer cleanup()

	// Create test namespaces
	testNamespaces := []string{"edge-ns-1", "edge-ns-2"}
	for _, ns := range testNamespaces {
		_, err := clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	// Create test deployments
	for _, namespace := range testNamespaces {
		_, err := clientset.AppsV1().Deployments(namespace).Create(context.Background(), &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test-deployment",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "test-deployment",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "nginx",
								Image: "nginx:1.21",
							},
						},
					},
				},
			},
		}, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	// Create informer manager
	informerManager := informer.NewDeploymentInformerManager(clientset)

	t.Run("SingleNamespace", func(t *testing.T) {
		// Test with single namespace (no commas)
		namespaceString := "edge-ns-1"
		namespacesToWatch := strings.Split(namespaceString, ",")
		for i, ns := range namespacesToWatch {
			namespacesToWatch[i] = strings.TrimSpace(ns)
		}

		// Start informer for single namespace
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		informerManager.StartInformer(ctx, namespacesToWatch[0])

		// Wait for informer to sync
		time.Sleep(1 * time.Second)

		// Create handler manager
		handlerManager := NewHandlerManager(informerManager, "test-version-1.0.0")

		// Test namespaces endpoint
		ctx2 := &fasthttp.RequestCtx{}
		ctx2.Request.SetRequestURI("/namespaces")
		ctx2.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx2)

		assert.Equal(t, 200, ctx2.Response.StatusCode())

		var response NamespaceResponse
		err := json.Unmarshal(ctx2.Response.Body(), &response)
		require.NoError(t, err)

		assert.Equal(t, 1, response.Count)
		assert.Contains(t, response.Namespaces, "edge-ns-1")
	})

	t.Run("NamespacesWithSpaces", func(t *testing.T) {
		// Test with namespaces that have spaces around commas
		namespaceString := "edge-ns-1 , edge-ns-2"
		namespacesToWatch := strings.Split(namespaceString, ",")
		for i, ns := range namespacesToWatch {
			namespacesToWatch[i] = strings.TrimSpace(ns)
		}

		// Start informers for both namespaces
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		for _, namespace := range namespacesToWatch {
			informerManager.StartInformer(ctx, namespace)
		}

		// Wait for informers to sync
		time.Sleep(1 * time.Second)

		// Create handler manager
		handlerManager := NewHandlerManager(informerManager, "test-version-1.0.0")

		// Test namespaces endpoint
		ctx2 := &fasthttp.RequestCtx{}
		ctx2.Request.SetRequestURI("/namespaces")
		ctx2.Request.Header.SetMethod("GET")

		handler := handlerManager.CreateHandler()
		handler(ctx2)

		assert.Equal(t, 200, ctx2.Response.StatusCode())

		var response NamespaceResponse
		err := json.Unmarshal(ctx2.Response.Body(), &response)
		require.NoError(t, err)

		assert.Equal(t, 2, response.Count)
		assert.Contains(t, response.Namespaces, "edge-ns-1")
		assert.Contains(t, response.Namespaces, "edge-ns-2")
	})
}
