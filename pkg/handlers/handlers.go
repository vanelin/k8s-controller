package handlers

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"github.com/vanelin/k8s-controller.git/pkg/informer"
)

// DeploymentResponse represents the response structure for deployment endpoints
type DeploymentResponse struct {
	Namespace   string   `json:"namespace"`
	Deployments []string `json:"deployments"`
	Count       int      `json:"count"`
}

// NamespaceResponse represents the response structure for namespace endpoints
type NamespaceResponse struct {
	Namespaces []string `json:"namespaces"`
	Count      int      `json:"count"`
}

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// DeploymentsAllResponse represents the response for all deployments across namespaces
type DeploymentsAllResponse struct {
	Namespaces []DeploymentResponse `json:"namespaces"`
	TotalCount int                  `json:"total_count"`
}

// HandlerManager manages HTTP handlers with access to the informer manager
type HandlerManager struct {
	informerManager *informer.DeploymentInformerManager
	appVersion      string
}

// NewHandlerManager creates a new handler manager
func NewHandlerManager(informerManager *informer.DeploymentInformerManager, appVersion string) *HandlerManager {
	return &HandlerManager{
		informerManager: informerManager,
		appVersion:      appVersion,
	}
}

// CreateHandler creates the main HTTP handler with routing
func (hm *HandlerManager) CreateHandler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		requestID := uuid.New().String()
		ctx.Response.Header.Set("X-Request-ID", requestID)

		logger := log.With().Str("request_id", requestID).Logger()

		path := string(ctx.Path())
		method := string(ctx.Method())

		logger.Info().Str("method", method).Str("path", path).Msg("HTTP request received")

		switch {
		case path == "/deployments" && method == "GET":
			hm.handleGetDeployments(ctx, logger)
		case strings.HasPrefix(path, "/deployments/") && method == "GET":
			hm.handleGetDeploymentsByNamespace(ctx, logger)
		case path == "/namespaces" && method == "GET":
			hm.handleGetNamespaces(ctx, logger)
		case path == "/" && method == "GET":
			hm.handleRoot(ctx, logger)
		default:
			hm.handleNotFound(ctx, logger)
		}
	}
}

// handleGetDeployments handles GET /deployments - returns deployments from all watched namespaces
func (hm *HandlerManager) handleGetDeployments(ctx *fasthttp.RequestCtx, logger zerolog.Logger) {
	logger.Info().Msg("Deployments request received")

	availableNamespaces := hm.informerManager.GetAvailableNamespaces()
	if len(availableNamespaces) == 0 {
		hm.writeErrorResponse(ctx, "No namespaces are being watched", 404, logger)
		return
	}

	var responses []DeploymentResponse
	total := 0
	for _, ns := range availableNamespaces {
		deployments := hm.informerManager.GetDeploymentNames(ns)
		resp := DeploymentResponse{
			Namespace:   ns,
			Deployments: deployments,
			Count:       len(deployments),
		}
		responses = append(responses, resp)
		total += len(deployments)
	}

	allResp := DeploymentsAllResponse{
		Namespaces: responses,
		TotalCount: total,
	}

	hm.writeJSONResponse(ctx, allResp, 200, logger)
}

// handleGetDeploymentsByNamespace handles GET /deployments/{namespace} - returns deployments from specific namespace
func (hm *HandlerManager) handleGetDeploymentsByNamespace(ctx *fasthttp.RequestCtx, logger zerolog.Logger) {
	path := string(ctx.Path())
	parts := strings.Split(path, "/")

	if len(parts) != 3 {
		hm.writeErrorResponse(ctx, "Invalid path format. Use /deployments/{namespace}", 400, logger)
		return
	}

	namespace := parts[2]
	// URL decode the namespace
	decodedNamespace, err := url.QueryUnescape(namespace)
	if err != nil {
		hm.writeErrorResponse(ctx, "Invalid namespace encoding", 400, logger)
		return
	}

	logger.Info().Str("namespace", decodedNamespace).Msg("Deployments by namespace request received")

	// Check if informer exists for this namespace
	if !hm.informerManager.HasInformer(decodedNamespace) {
		hm.writeErrorResponse(ctx, "Namespace not being watched: "+decodedNamespace, 404, logger)
		return
	}

	deployments := hm.informerManager.GetDeploymentNames(decodedNamespace)

	response := DeploymentResponse{
		Namespace:   decodedNamespace,
		Deployments: deployments,
		Count:       len(deployments),
	}

	hm.writeJSONResponse(ctx, response, 200, logger)
}

// handleGetNamespaces handles GET /namespaces - returns list of available namespaces
func (hm *HandlerManager) handleGetNamespaces(ctx *fasthttp.RequestCtx, logger zerolog.Logger) {
	logger.Info().Msg("Namespaces request received")

	namespaces := hm.informerManager.GetAvailableNamespaces()

	response := NamespaceResponse{
		Namespaces: namespaces,
		Count:      len(namespaces),
	}

	hm.writeJSONResponse(ctx, response, 200, logger)
}

// handleRoot handles GET / - returns basic API information
func (hm *HandlerManager) handleRoot(ctx *fasthttp.RequestCtx, logger zerolog.Logger) {
	logger.Info().Msg("Root request received")

	response := map[string]interface{}{
		"message": "Kubernetes Controller API",
		"version": hm.appVersion,
		"endpoints": map[string]string{
			"deployments": "/deployments",
			"namespaces":  "/namespaces",
		},
	}

	hm.writeJSONResponse(ctx, response, 200, logger)
}

// handleNotFound handles 404 responses
func (hm *HandlerManager) handleNotFound(ctx *fasthttp.RequestCtx, logger zerolog.Logger) {
	logger.Warn().Str("path", string(ctx.Path())).Msg("Endpoint not found")

	response := ErrorResponse{
		Error:   "Not Found",
		Message: "The requested endpoint does not exist",
	}

	hm.writeJSONResponse(ctx, response, 404, logger)
}

// writeJSONResponse writes a JSON response to the HTTP context
func (hm *HandlerManager) writeJSONResponse(ctx *fasthttp.RequestCtx, data interface{}, statusCode int, logger zerolog.Logger) {
	ctx.SetStatusCode(statusCode)
	ctx.Response.Header.Set("Content-Type", "application/json")

	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to marshal JSON response")
		ctx.SetStatusCode(500)
		ctx.Response.Header.Set("Content-Type", "application/json")
		if _, werr := ctx.WriteString(`{"error":"Internal Server Error","message":"Failed to serialize response"}`); werr != nil {
			logger.Error().Err(werr).Msg("Failed to write error response")
		}
		return
	}

	if _, werr := ctx.Write(jsonData); werr != nil {
		logger.Error().Err(werr).Msg("Failed to write JSON response")
	}
	logger.Info().Int("status_code", statusCode).Msg("Response sent successfully")
}

// writeErrorResponse writes an error JSON response to the HTTP context
func (hm *HandlerManager) writeErrorResponse(ctx *fasthttp.RequestCtx, message string, statusCode int, logger zerolog.Logger) {
	response := ErrorResponse{
		Error:   "Request Error",
		Message: message,
	}

	hm.writeJSONResponse(ctx, response, statusCode, logger)
}
