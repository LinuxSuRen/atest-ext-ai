package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"atest-ext-ai-core/internal/ai"
	"atest-ext-ai-core/internal/config"
	"atest-ext-ai-core/internal/errors"
	"atest-ext-ai-core/internal/logger"
	"atest-ext-ai-core/pkg/models"
	pb "github.com/linuxsuren/api-testing/pkg/server"
)

// AIPluginServer implements the RunnerExtension interface
type AIPluginServer struct {
	pb.UnimplementedRunnerExtensionServer
	config    *config.Config
	aiService ai.AIService
	startTime time.Time
}

// NewAIPluginServer creates a new AI plugin server instance
func NewAIPluginServer(cfg *config.Config, aiService ai.AIService) *AIPluginServer {
	return &AIPluginServer{
		config:    cfg,
		aiService: aiService,
		startTime: time.Now(),
	}
}

// Run implements the RunnerExtension interface
// This is the main entry point for AI plugin operations
func (s *AIPluginServer) Run(ctx context.Context, req *pb.TestSuiteWithCase) (*pb.CommonResult, error) {
	logger.Debugf("Received gRPC request for AI plugin")

	// Parse the request to determine the action
	var aiReq models.AIRequest
	if req.Case != nil && req.Case.Request != nil {
		// Try to parse the request body as JSON to get AI-specific parameters
		if err := json.Unmarshal([]byte(req.Case.Request.Body), &aiReq); err != nil {
			logger.Warnf("Failed to parse request body as JSON, using default action: %v", err)
			// If parsing fails, create a default request
			aiReq = models.AIRequest{
				Action: "convert_to_sql", // Default action
				Data:   map[string]interface{}{"query": req.Case.Request.Body},
			}
		}
	} else {
		appErr := errors.ErrInvalidGRPCRequest("missing case or request data")
		logger.ErrorWithErr("Invalid gRPC request received", appErr)
		return &pb.CommonResult{
			Success: false,
			Message: appErr.Error(),
		}, nil
	}

	logger.Infof("Processing AI request with action: %s", aiReq.Action)

	// Route to appropriate handler based on action
	switch aiReq.Action {
	case "convert_to_sql":
		return s.handleConvertToSQL(ctx, &aiReq)
	case "ping":
		return s.handlePing(ctx, &aiReq)
	case "health_check":
		return s.handleHealthCheck(ctx, &aiReq)
	case "get_model_info":
		return s.handleGetModelInfo(ctx, &aiReq)
	default:
		appErr := errors.ErrInvalidGRPCRequest(fmt.Sprintf("unknown action: %s", aiReq.Action))
		logger.ErrorWithErr("Unknown action requested", appErr)
		return &pb.CommonResult{
			Success: false,
			Message: appErr.Error(),
		}, nil
	}
}

// handleConvertToSQL handles SQL conversion requests
func (s *AIPluginServer) handleConvertToSQL(ctx context.Context, req *models.AIRequest) (*pb.CommonResult, error) {
	// Extract query from Data field
	var query string
	if req.Data != nil {
		if q, ok := req.Data["query"].(string); ok {
			query = q
		}
	}

	logger.Debugf("Handling ConvertToSQL request for query: %s", query)

	if query == "" {
		appErr := errors.ErrInvalidInputData("query cannot be empty")
		logger.ErrorWithErr("Empty query provided for SQL conversion", appErr)
		return &pb.CommonResult{
			Success: false,
			Message: appErr.Error(),
		}, nil
	}

	// Create SQL conversion request
	sqlReq := &models.SQLConversionRequest{
		Query:   query,
		Context: "API testing context",
		Dialect: "mysql", // Default dialect
	}

	// Call AI service to convert to SQL
	sqlResp, err := s.aiService.ConvertToSQL(ctx, sqlReq)
	if err != nil {
		logger.ErrorfWithErr(err, "Failed to convert query to SQL")
		return &pb.CommonResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	logger.Infof("Successfully converted query to SQL")

	return &pb.CommonResult{
		Success: true,
		Message: sqlResp.SQL,
	}, nil
}

// handlePing handles ping requests
func (s *AIPluginServer) handlePing(ctx context.Context, req *models.AIRequest) (*pb.CommonResult, error) {
	logger.Debug("Handling ping request")
	return &pb.CommonResult{
		Success: true,
		Message: "pong",
	}, nil
}

// handleHealthCheck handles health check requests
func (s *AIPluginServer) handleHealthCheck(ctx context.Context, req *models.AIRequest) (*pb.CommonResult, error) {
	logger.Debug("Handling health check request")

	// Check if AI service is healthy
	if s.aiService == nil {
		appErr := errors.ErrAIServiceUnavailable("AI service not initialized")
		logger.ErrorWithErr("Health check failed", appErr)
		return &pb.CommonResult{
			Success: false,
			Message: appErr.Error(),
		}, nil
	}

	logger.Info("Health check passed")
	return &pb.CommonResult{
		Success: true,
		Message: "AI plugin is healthy",
	}, nil
}

// handleGetModelInfo handles model information requests
func (s *AIPluginServer) handleGetModelInfo(ctx context.Context, req *models.AIRequest) (*pb.CommonResult, error) {
	logger.Debug("Handling get model info request")

	// Return mock model information for MVP
	modelInfo := models.ModelInfo{
		Name:         "mock-ai-model",
		Version:      "1.0.0",
		Provider:     "mock-provider",
		Capabilities: []string{"sql_conversion", "text_analysis"},
	}

	// Convert to JSON
	modelInfoJSON, err := json.Marshal(modelInfo)
	if err != nil {
		appErr := errors.ErrInvalidAIResponse("failed to serialize model info")
		logger.ErrorfWithErr(err, "Failed to marshal model info to JSON")
		return &pb.CommonResult{
			Success: false,
			Message: appErr.Error(),
		}, nil
	}

	logger.Info("Successfully retrieved model info")
	return &pb.CommonResult{
		Success: true,
		Message: string(modelInfoJSON),
	}, nil
}

// RegisterAIPluginServer registers the AI plugin server with gRPC
func RegisterAIPluginServer(s *grpc.Server, srv *AIPluginServer) {
	pb.RegisterRunnerExtensionServer(s, srv)
	logger.Info("AI Plugin Server registered successfully")
}

// StartGRPCServer starts the gRPC server
func StartGRPCServer(cfg *config.Config, aiService ai.AIService) error {
	logger.Infof("Starting gRPC server on port %d", cfg.Server.Port)

	// Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		appErr := errors.ErrGRPCServerStartFailure(err)
		logger.ErrorfWithErr(err, "Failed to listen on port %d", cfg.Server.Port)
		return appErr
	}

	// Create gRPC server
	s := grpc.NewServer()

	// Create and register AI plugin server
	aiPluginServer := NewAIPluginServer(cfg, aiService)
	RegisterAIPluginServer(s, aiPluginServer)

	// Create and register UI extension server
	uiExtensionServer := NewUIExtensionServer()
	RegisterUIExtensionServer(s, uiExtensionServer)

	// Enable reflection for debugging
	reflection.Register(s)
	logger.Info("gRPC reflection enabled for debugging")

	logger.Infof("AI Plugin gRPC server started successfully on port %d", cfg.Server.Port)
	return s.Serve(lis)
}
