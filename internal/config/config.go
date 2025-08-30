package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	AI       AIConfig       `yaml:"ai"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// AIConfig represents AI model configuration
type AIConfig struct {
	DefaultModel string                 `yaml:"default_model"`
	Models       map[string]ModelConfig `yaml:"models"`
	Cache        CacheConfig            `yaml:"cache"`
}

// ModelConfig represents individual AI model configuration
type ModelConfig struct {
	Provider    string            `yaml:"provider"`
	APIKey      string            `yaml:"api_key"`
	Endpoint    string            `yaml:"endpoint"`
	MaxTokens   int               `yaml:"max_tokens"`
	Temperature float64           `yaml:"temperature"`
	Timeout     time.Duration     `yaml:"timeout"`
	Headers     map[string]string `yaml:"headers"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled bool          `yaml:"enabled"`
	TTL     time.Duration `yaml:"ttl"`
	Size    int           `yaml:"size"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "localhost"),
			Port: getEnvAsInt("SERVER_PORT", 50051),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			DBName:   getEnv("DB_NAME", "ai_plugin"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		AI: AIConfig{
			DefaultModel: getEnv("AI_MODEL", "gpt-3.5-turbo"),
			Cache: CacheConfig{
				Enabled: getEnvAsBool("AI_CACHE_ENABLED", true),
				TTL:     getEnvAsDuration("AI_CACHE_TTL", time.Hour),
				Size:    getEnvAsInt("AI_CACHE_SIZE", 1000),
			},
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "text"),
			Output: getEnv("LOG_OUTPUT", "stdout"),
		},
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server configuration
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Validate database configuration
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user cannot be empty")
	}

	// Validate AI configuration
	if c.AI.DefaultModel == "" {
		return fmt.Errorf("AI default model cannot be empty")
	}

	// Validate logging configuration
	validLevels := []string{"debug", "info", "warn", "error", "fatal"}
	if !contains(validLevels, strings.ToLower(c.Logging.Level)) {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	validFormats := []string{"text", "json"}
	if !contains(validFormats, strings.ToLower(c.Logging.Format)) {
		return fmt.Errorf("invalid log format: %s", c.Logging.Format)
	}

	return nil
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
