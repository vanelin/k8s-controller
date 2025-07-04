package handlers

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/vanelin/k8s-controller/pkg/informer"
)

func TestNewHandlerManager(t *testing.T) {
	informerManager := informer.NewDeploymentInformerManager(nil)
	handlerManager := NewHandlerManager(informerManager, "test-version")

	assert.NotNil(t, handlerManager)
	assert.Equal(t, informerManager, handlerManager.informerManager)
	assert.Equal(t, "test-version", handlerManager.appVersion)
}

func TestHandlerManager_CreateHandler(t *testing.T) {
	informerManager := informer.NewDeploymentInformerManager(nil)
	handlerManager := NewHandlerManager(informerManager, "test-version")

	handler := handlerManager.CreateHandler()
	assert.NotNil(t, handler)
}

func TestHandlerManager_handleRoot(t *testing.T) {
	informerManager := informer.NewDeploymentInformerManager(nil)
	handlerManager := NewHandlerManager(informerManager, "v1.2.3")

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/")
	ctx.Request.Header.SetMethod("GET")

	logger := zerolog.Nop()
	handlerManager.handleRoot(ctx, logger)

	assert.Equal(t, 200, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Header.ContentType()), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(ctx.Response.Body(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Kubernetes Controller API", response["message"])
	assert.Equal(t, "v1.2.3", response["version"])
}

func TestHandlerManager_handleGetNamespaces(t *testing.T) {
	informerManager := informer.NewDeploymentInformerManager(nil)
	handlerManager := NewHandlerManager(informerManager, "test-version")

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/namespaces")
	ctx.Request.Header.SetMethod("GET")

	logger := zerolog.Nop()
	handlerManager.handleGetNamespaces(ctx, logger)

	assert.Equal(t, 200, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Header.ContentType()), "application/json")

	var response NamespaceResponse
	err := json.Unmarshal(ctx.Response.Body(), &response)
	require.NoError(t, err)

	assert.Equal(t, 0, response.Count)
	assert.Empty(t, response.Namespaces)
}

func TestHandlerManager_handleGetDeployments(t *testing.T) {
	informerManager := informer.NewDeploymentInformerManager(nil)
	handlerManager := NewHandlerManager(informerManager, "test-version")

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/deployments")
	ctx.Request.Header.SetMethod("GET")

	logger := zerolog.Nop()
	handlerManager.handleGetDeployments(ctx, logger)

	assert.Equal(t, 404, ctx.Response.StatusCode())

	var response ErrorResponse
	err := json.Unmarshal(ctx.Response.Body(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Request Error", response.Error)
	assert.Contains(t, response.Message, "No namespaces are being watched")
}

func TestHandlerManager_handleGetDeploymentsByNamespace_InvalidPath(t *testing.T) {
	informerManager := informer.NewDeploymentInformerManager(nil)
	handlerManager := NewHandlerManager(informerManager, "test-version")

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/deployments/invalid/path")
	ctx.Request.Header.SetMethod("GET")

	logger := zerolog.Nop()
	handlerManager.handleGetDeploymentsByNamespace(ctx, logger)

	assert.Equal(t, 400, ctx.Response.StatusCode())

	var response ErrorResponse
	err := json.Unmarshal(ctx.Response.Body(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Request Error", response.Error)
	assert.Contains(t, response.Message, "Invalid path format")
}

func TestHandlerManager_handleGetDeploymentsByNamespace_NamespaceNotWatched(t *testing.T) {
	informerManager := informer.NewDeploymentInformerManager(nil)
	handlerManager := NewHandlerManager(informerManager, "test-version")

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/deployments/test-namespace")
	ctx.Request.Header.SetMethod("GET")

	logger := zerolog.Nop()
	handlerManager.handleGetDeploymentsByNamespace(ctx, logger)

	assert.Equal(t, 404, ctx.Response.StatusCode())

	var response ErrorResponse
	err := json.Unmarshal(ctx.Response.Body(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Request Error", response.Error)
	assert.Contains(t, response.Message, "Namespace not being watched")
}

func TestHandlerManager_handleNotFound(t *testing.T) {
	informerManager := informer.NewDeploymentInformerManager(nil)
	handlerManager := NewHandlerManager(informerManager, "test-version")

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/unknown-endpoint")
	ctx.Request.Header.SetMethod("GET")

	logger := zerolog.Nop()
	handlerManager.handleNotFound(ctx, logger)

	assert.Equal(t, 404, ctx.Response.StatusCode())

	var response ErrorResponse
	err := json.Unmarshal(ctx.Response.Body(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Not Found", response.Error)
	assert.Equal(t, "The requested endpoint does not exist", response.Message)
}

func TestHandlerManager_writeJSONResponse(t *testing.T) {
	informerManager := informer.NewDeploymentInformerManager(nil)
	handlerManager := NewHandlerManager(informerManager, "test-version")

	ctx := &fasthttp.RequestCtx{}

	testData := map[string]string{
		"test": "data",
	}

	logger := zerolog.Nop()
	handlerManager.writeJSONResponse(ctx, testData, 200, logger)

	assert.Equal(t, 200, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Header.ContentType()), "application/json")

	var response map[string]string
	err := json.Unmarshal(ctx.Response.Body(), &response)
	require.NoError(t, err)

	assert.Equal(t, "data", response["test"])
}

func TestHandlerManager_writeErrorResponse(t *testing.T) {
	informerManager := informer.NewDeploymentInformerManager(nil)
	handlerManager := NewHandlerManager(informerManager, "test-version")

	ctx := &fasthttp.RequestCtx{}

	logger := zerolog.Nop()
	handlerManager.writeErrorResponse(ctx, "Test error message", 400, logger)

	assert.Equal(t, 400, ctx.Response.StatusCode())

	var response ErrorResponse
	err := json.Unmarshal(ctx.Response.Body(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Request Error", response.Error)
	assert.Equal(t, "Test error message", response.Message)
}

func TestURLDecoding(t *testing.T) {
	// Test URL encoding/decoding for namespaces with special characters
	testCases := []struct {
		encoded   string
		decoded   string
		shouldErr bool
	}{
		{"default", "default", false},
		{"kube-system", "kube-system", false},
		{"my-namespace", "my-namespace", false},
		{"namespace%20with%20spaces", "namespace with spaces", false},
		{"namespace%2Fwith%2Fslashes", "namespace/with/slashes", false},
	}

	for _, tc := range testCases {
		t.Run(tc.encoded, func(t *testing.T) {
			decoded, err := url.QueryUnescape(tc.encoded)
			if tc.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.decoded, decoded)
			}
		})
	}
}
