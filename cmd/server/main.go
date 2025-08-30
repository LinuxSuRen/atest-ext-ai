package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"atest-ext-ai-core/internal/ai"
	"atest-ext-ai-core/internal/config"
	"atest-ext-ai-core/internal/errors"
	"atest-ext-ai-core/internal/logger"
	"atest-ext-ai-core/internal/server"
)

const (
	// DefaultPort is the default port for the gRPC server
	DefaultPort = "50051"
	// DefaultConfigPath is the default configuration file path
	DefaultConfigPath = "config.yaml"
	// GracefulShutdownTimeout is the timeout for graceful shutdown
	GracefulShutdownTimeout = 30 * time.Second
)

func main() {
	// Initialize logger first
	logger.InitGlobalLogger(&config.LoggingConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	})

	logger.Info("Starting AI Plugin Server...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.FatalWithErr("Failed to load configuration", err)
	}

	// Re-initialize logger with loaded configuration
	logger.InitGlobalLogger(&cfg.Logging)
	logger.Info("Logger re-initialized with configuration")

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		appErr := errors.ErrConfigLoadFailure(err)
		logger.FatalWithErr("Configuration validation failed", appErr)
	}

	logger.Infof("Configuration loaded successfully. Server will listen on port %d", cfg.Server.Port)

	// Initialize AI service
	aiService, err := ai.NewService(cfg)
	if err != nil {
		logger.FatalWithErr("Failed to initialize AI service", err)
	}
	defer func() {
		if err := aiService.Close(); err != nil {
			logger.ErrorWithErr("Error closing AI service", err)
		}
	}()

	logger.Info("AI service initialized successfully")

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(4*1024*1024), // 4MB
		grpc.MaxSendMsgSize(4*1024*1024), // 4MB
	)

	// Register AI plugin server
	aiPluginServer := server.NewAIPluginServer(cfg, aiService)
	server.RegisterAIPluginServer(grpcServer, aiPluginServer)

	// Enable reflection for development (always enabled in MVP)
	reflection.Register(grpcServer)
	logger.Info("gRPC reflection enabled")

	// Setup network listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		logger.FatalWithErr(fmt.Sprintf("Failed to listen on port %d", cfg.Server.Port), err)
	}

	logger.Infof("AI Plugin Server listening on :%d", cfg.Server.Port)

	// Start server in a goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			logger.FatalWithErr("Failed to serve gRPC server", err)
		}
	}()

	logger.Info("AI Plugin Server started successfully")

	// Wait for interrupt signal to gracefully shutdown the server
	waitForShutdown(grpcServer, aiService)
}

// loadConfig loads configuration from environment variables and config file
func loadConfig() (*config.Config, error) {
	// Try to load from config file first
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	// For now, just use the Load function which loads from environment variables
	cfg, err := config.Load()
	if err != nil {
		// If config loading fails, create default config
		// Note: Using fmt.Printf here as logger may not be initialized yet
		fmt.Printf("Config loading failed, using default configuration: %v\n", err)
		cfg = createDefaultConfig()
	}

	// Override with environment variables
	overrideWithEnvVars(cfg)

	return cfg, nil
}

// createDefaultConfig creates a default configuration
func createDefaultConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port: getEnvAsIntOrDefault("SERVER_PORT", 50051),
			Host: getEnvOrDefault("SERVER_HOST", "localhost"),
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			DBName:   "ai_plugin",
			User:     "postgres",
			Password: "",
			SSLMode:  "disable",
		},
		AI: config.AIConfig{
			DefaultModel: "mock",
			Models: map[string]config.ModelConfig{
				"mock": {
					Provider:    "mock",
					MaxTokens:   4096,
					Temperature: 0.7,
					Timeout:     30 * time.Second,
				},
			},
			Cache: config.CacheConfig{
				Enabled: true,
				Size:    1000,
				TTL:     10 * time.Minute,
			},
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// overrideWithEnvVars overrides configuration with environment variables
func overrideWithEnvVars(cfg *config.Config) {
	port := getEnvAsIntOrDefault("SERVER_PORT", 50051)
	if port != 0 {
		cfg.Server.Port = port
	}
	if host := os.Getenv("SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if model := os.Getenv("AI_DEFAULT_MODEL"); model != "" {
		cfg.AI.DefaultModel = model
	}
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.Logging.Level = level
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsIntOrDefault returns environment variable as int or default
func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// waitForShutdown waits for interrupt signal and gracefully shuts down the server
func waitForShutdown(grpcServer *grpc.Server, aiService *ai.Service) {
	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)

	// Register the channel to receive specific signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-sigChan
	fmt.Printf("Received signal: %v. Initiating graceful shutdown...\n", sig)

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), GracefulShutdownTimeout)
	defer cancel()

	// Channel to signal when shutdown is complete
	shutdownComplete := make(chan struct{})

	go func() {
		// Gracefully stop the gRPC server
		fmt.Println("Stopping gRPC server...")
		grpcServer.GracefulStop()

		// Close AI service
		fmt.Println("Closing AI service...")
		if err := aiService.Close(); err != nil {
			fmt.Printf("Error closing AI service: %v\n", err)
		}

		fmt.Println("Shutdown complete")
		close(shutdownComplete)
	}()

	// Wait for shutdown to complete or timeout
	select {
	case <-shutdownComplete:
		fmt.Println("Server shutdown gracefully")
	case <-ctx.Done():
		fmt.Println("Shutdown timeout exceeded, forcing exit")
		grpcServer.Stop() // Force stop
	}

	fmt.Println("AI Plugin Server stopped")
}
