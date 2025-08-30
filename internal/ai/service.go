package ai

import (
	"context"
	"fmt"
	"log"
	"time"

	"atest-ext-ai-core/internal/config"
	"atest-ext-ai-core/internal/errors"
	"atest-ext-ai-core/internal/logger"
	"atest-ext-ai-core/pkg/models"
)

// AIService represents the AI service interface
type AIService interface {
	ConvertToSQL(ctx context.Context, req *models.SQLConversionRequest) (*models.SQLConversionResponse, error)
	GetModelInfo(ctx context.Context, modelName string) (*models.ModelInfo, error)
	IsHealthy(ctx context.Context) bool
	Close() error
}

// AIClient represents an AI client interface
type AIClient interface {
	ConvertToSQL(ctx context.Context, req *models.SQLConversionRequest) (*models.SQLConversionResponse, error)
	GetModelInfo(ctx context.Context) (*models.ModelInfo, error)
	IsHealthy(ctx context.Context) bool
	Close() error
}

// Service implements the AIService interface
type Service struct {
	config  *config.Config
	clients map[string]AIClient
	cache   *Cache
}

// NewService creates a new AI service instance
func NewService(cfg *config.Config) (*Service, error) {
	logger.Info("Initializing AI service")

	if cfg == nil {
		appErr := errors.ErrInvalidConfig("config cannot be nil")
		logger.ErrorWithErr("Failed to create AI service", appErr)
		return nil, appErr
	}

	service := &Service{
		config:  cfg,
		clients: make(map[string]AIClient),
	}

	// Initialize cache (always enabled for MVP)
	service.cache = NewCache(100, time.Hour) // Default cache settings
	logger.Debug("Cache initialized with default settings")

	// Initialize AI clients for each configured model
	for modelName, modelConfig := range cfg.AI.Models {
		client, err := NewAIClient(modelName, &modelConfig)
		if err != nil {
			log.Printf("Failed to initialize client for model %s: %v", modelName, err)
			continue
		}
		service.clients[modelName] = client
		log.Printf("Initialized AI client for model: %s", modelName)
	}

	// MVP: If no models are configured, create a default mock client
	if len(service.clients) == 0 {
		logger.Info("No models configured, creating default mock client for MVP")
		mockConfig := &config.ModelConfig{
			Provider:  "mock",
			MaxTokens: 4096,
			Timeout:   30,
		}
		client, err := NewAIClient(cfg.AI.DefaultModel, mockConfig)
		if err != nil {
			appErr := errors.ErrAIServiceUnavailable(fmt.Sprintf("failed to create default mock client: %v", err))
			logger.ErrorWithErr("Failed to create default mock client", appErr)
			return nil, appErr
		}
		// Register both with default model name and "mock" for compatibility
		service.clients[cfg.AI.DefaultModel] = client
		service.clients["mock"] = client
		log.Printf("Created default mock client for model: %s", cfg.AI.DefaultModel)
	}

	logger.Info("AI service initialized successfully")
	return service, nil
}

// ConvertToSQL converts natural language to SQL using the configured AI model
func (s *Service) ConvertToSQL(ctx context.Context, req *models.SQLConversionRequest) (*models.SQLConversionResponse, error) {
	logger.Debugf("Converting query to SQL: %s", req.Query)

	// Check cache first if enabled
	if s.cache != nil {
		cacheKey := s.generateCacheKey("sql", req.Query, req.Context, req.Dialect)
		if cached := s.cache.Get(cacheKey); cached != nil {
			if response, ok := cached.(*models.SQLConversionResponse); ok {
				logger.Debug("SQL result found in cache")
				log.Printf("Cache hit for SQL conversion: %s", req.Query)
				return response, nil
			}
		}
	}

	// Get the default model client
	client, exists := s.clients[s.config.AI.DefaultModel]
	if !exists {
		appErr := errors.ErrAIServiceUnavailable(fmt.Sprintf("default model client not found: %s", s.config.AI.DefaultModel))
		logger.ErrorWithErr("AI client not found", appErr)
		return nil, appErr
	}

	// Call the AI client
	response, err := client.ConvertToSQL(ctx, req)
	if err != nil {
		appErr := errors.ErrAIServiceUnavailable(fmt.Sprintf("AI client error: %v", err))
		logger.ErrorWithErr("AI client conversion failed", appErr)
		return nil, appErr
	}

	logger.Debugf("SQL conversion completed: %s", response.SQL)

	// Cache the response if caching is enabled
	if s.cache != nil {
		cacheKey := s.generateCacheKey("sql", req.Query, req.Context, req.Dialect)
		s.cache.Set(cacheKey, response)
		logger.Debug("SQL result cached successfully")
	}

	return response, nil
}

// GetModelInfo returns information about the specified model
func (s *Service) GetModelInfo(ctx context.Context, modelName string) (*models.ModelInfo, error) {
	if modelName == "" {
		modelName = s.config.AI.DefaultModel
	}

	client, exists := s.clients[modelName]
	if !exists {
		return nil, fmt.Errorf("model client not found: %s", modelName)
	}

	return client.GetModelInfo(ctx)
}

// IsHealthy checks if the AI service is healthy
func (s *Service) IsHealthy(ctx context.Context) bool {
	// Check if at least one client is healthy
	for modelName, client := range s.clients {
		if client.IsHealthy(ctx) {
			log.Printf("Model %s is healthy", modelName)
			return true
		}
	}
	return false
}

// Close closes all AI clients and cleans up resources
func (s *Service) Close() error {
	var lastErr error
	for modelName, client := range s.clients {
		if err := client.Close(); err != nil {
			log.Printf("Error closing client for model %s: %v", modelName, err)
			lastErr = err
		}
	}
	return lastErr
}

// generateCacheKey generates a cache key for the given parameters
func (s *Service) generateCacheKey(operation, query, context, dialect string) string {
	return fmt.Sprintf("%s:%s:%s:%s", operation, query, context, dialect)
}

// MockAIClient implements AIClient for MVP testing
type MockAIClient struct {
	modelName   string
	modelConfig *config.ModelConfig
	startTime   time.Time
}

// NewAIClient creates a new AI client (MVP: returns mock client)
func NewAIClient(modelName string, modelConfig *config.ModelConfig) (AIClient, error) {
	// MVP implementation: return mock client
	return &MockAIClient{
		modelName:   modelName,
		modelConfig: modelConfig,
		startTime:   time.Now(),
	}, nil
}

// ConvertToSQL implements AIClient.ConvertToSQL (MVP: returns fixed response)
func (c *MockAIClient) ConvertToSQL(ctx context.Context, req *models.SQLConversionRequest) (*models.SQLConversionResponse, error) {
	logger.Debugf("Mock AI client processing query: %s", req.Query)
	log.Printf("MockAIClient: Converting query '%s' to SQL", req.Query)

	// MVP: Return fixed SQL based on common patterns
	var sql string
	var explanation string

	switch {
	case contains(req.Query, "user", "users"):
		sql = "SELECT * FROM users"
		explanation = "Generated SQL to query users table based on natural language input"
	case contains(req.Query, "order", "orders"):
		sql = "SELECT * FROM orders"
		explanation = "Generated SQL to query orders table based on natural language input"
	case contains(req.Query, "product", "products"):
		sql = "SELECT * FROM products"
		explanation = "Generated SQL to query products table based on natural language input"
	default:
		sql = "SELECT 1"
		explanation = "Default SQL query for unrecognized natural language input"
	}

	logger.Debugf("Mock AI client generated SQL: %s", sql)

	return &models.SQLConversionResponse{
		SQL:         sql,
		Success:     true,
		Confidence:  0.85,
		Explanation: explanation,
		Warnings:    []string{"This is a mock response for MVP testing"},
		Model:       c.modelName,
		Provider:    c.modelConfig.Provider,
	}, nil
}

// GetModelInfo implements AIClient.GetModelInfo
func (c *MockAIClient) GetModelInfo(ctx context.Context) (*models.ModelInfo, error) {
	return &models.ModelInfo{
		Name:     c.modelName,
		Provider: c.modelConfig.Provider,
		Version:  "mvp-1.0.0",
		Capabilities: []string{
			"sql_conversion",
			"natural_language_processing",
		},
		Limits: map[string]int{
			"max_tokens":          c.modelConfig.MaxTokens,
			"requests_per_minute": 60,
		},
		Metadata: map[string]string{
			"mode":        "mock",
			"description": "Mock AI client for MVP testing",
			"uptime":      time.Since(c.startTime).String(),
		},
	}, nil
}

// IsHealthy implements AIClient.IsHealthy
func (c *MockAIClient) IsHealthy(ctx context.Context) bool {
	return true // Mock client is always healthy
}

// Close implements AIClient.Close
func (c *MockAIClient) Close() error {
	log.Printf("Closing MockAIClient for model: %s", c.modelName)
	return nil
}

// contains checks if any of the keywords exist in the text (case-insensitive)
func contains(text string, keywords ...string) bool {
	text = fmt.Sprintf(" %s ", text)
	for _, keyword := range keywords {
		if len(text) >= len(keyword) {
			for i := 0; i <= len(text)-len(keyword); i++ {
				match := true
				for j, r := range keyword {
					if text[i+j] != byte(r) && text[i+j] != byte(r-32) && text[i+j] != byte(r+32) {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
	}
	return false
}
