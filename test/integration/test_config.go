/*
Copyright 2025 API Testing Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TestConfig holds configuration for integration tests
type TestConfig struct {
	// Plugin binary and configuration
	PluginBinaryPath string
	ConfigPath       string

	// Socket configuration
	SocketPath      string
	SocketTimeout   time.Duration

	// Test behavior
	StartupTimeout  time.Duration
	RequestTimeout  time.Duration
	LogLevel        string

	// Performance thresholds
	MaxCapabilitiesTime time.Duration
	MaxGenerationTime   time.Duration

	// Test data
	TestPrompts []string
	TestConfigs []string
}

// DefaultTestConfig returns a default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		SocketTimeout:       30 * time.Second,
		StartupTimeout:      30 * time.Second,
		RequestTimeout:      10 * time.Second,
		LogLevel:           "debug",
		MaxCapabilitiesTime: 5 * time.Second,
		MaxGenerationTime:   15 * time.Second,
		TestPrompts:        getDefaultTestPrompts(),
		TestConfigs:        getDefaultTestConfigs(),
	}
}

// LoadTestConfig loads test configuration from environment and defaults
func LoadTestConfig() (*TestConfig, error) {
	config := DefaultTestConfig()

	// Load from environment variables
	if path := os.Getenv("ATEST_EXT_AI_BINARY"); path != "" {
		config.PluginBinaryPath = path
	} else {
		// Try to find the binary
		binary := findPluginBinary()
		if binary == "" {
			return nil, fmt.Errorf("AI plugin binary not found")
		}
		config.PluginBinaryPath = binary
	}

	if path := os.Getenv("ATEST_EXT_AI_CONFIG"); path != "" {
		config.ConfigPath = path
	} else {
		config.ConfigPath = findConfigPath()
	}

	if path := os.Getenv("ATEST_EXT_AI_SOCKET"); path != "" {
		config.SocketPath = path
	} else {
		config.SocketPath = generateSocketPath()
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.LogLevel = level
	}

	return config, nil
}

// findPluginBinary locates the AI plugin binary
func findPluginBinary() string {
	// Look in the current project directory
	possiblePaths := []string{
		"../../atest-ext-ai",
		"../../../atest-ext-ai",
		"./atest-ext-ai",
		"/usr/local/bin/atest-ext-ai",
		"/opt/atest/bin/atest-ext-ai",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			abs, _ := filepath.Abs(path)
			return abs
		}
	}

	return ""
}

// findConfigPath locates the configuration file
func findConfigPath() string {
	possiblePaths := []string{
		"../../config/test.yaml",
		"../../config/development.yaml",
		"./config/test.yaml",
		"./config/development.yaml",
		"/etc/atest-ext-ai/config.yaml",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			abs, _ := filepath.Abs(path)
			return abs
		}
	}

	return "" // Use default configuration
}

// generateSocketPath generates a unique socket path for testing
func generateSocketPath() string {
	return fmt.Sprintf("/tmp/atest-ext-ai-integration-test-%d.sock", time.Now().UnixNano())
}

// getDefaultTestPrompts returns default test prompts for various scenarios
func getDefaultTestPrompts() []string {
	return []string{
		// Basic DDL
		"CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100), email VARCHAR(255))",
		"CREATE TABLE products (id INT, name VARCHAR(100), price DECIMAL(10,2))",

		// Basic DML
		"SELECT * FROM users WHERE active = 1",
		"INSERT INTO users (name, email) VALUES ('John Doe', 'john@example.com')",
		"UPDATE users SET last_login = NOW() WHERE id = 1",
		"DELETE FROM users WHERE created_at < NOW() - INTERVAL 30 DAY",

		// Complex queries
		"SELECT u.name, COUNT(o.id) as order_count FROM users u LEFT JOIN orders o ON u.id = o.user_id GROUP BY u.id ORDER BY order_count DESC",
		"SELECT p.name, SUM(oi.quantity * oi.price) as revenue FROM products p JOIN order_items oi ON p.id = oi.product_id GROUP BY p.id",

		// Schema operations
		"CREATE INDEX idx_users_email ON users(email)",
		"ALTER TABLE users ADD COLUMN phone VARCHAR(20)",
		"CREATE VIEW active_users AS SELECT * FROM users WHERE status = 'active'",

		// Natural language prompts
		"Create a table for storing blog posts with title, content, author, and timestamps",
		"Write a query to find the top 10 best-selling products this month",
		"Generate SQL to create a many-to-many relationship between users and roles",

		// Edge cases
		"SELECT 1", // Minimal query
		"CREATE DATABASE test_db", // Database creation
		"SHOW TABLES", // MySQL-specific
	}
}

// getDefaultTestConfigs returns default AI configuration options
func getDefaultTestConfigs() []string {
	return []string{
		`{}`, // Empty config
		`{"temperature": 0.7}`,
		`{"temperature": 0.3, "max_tokens": 500}`,
		`{"temperature": 0.9, "max_tokens": 1000}`,
		`{"temperature": 0.1, "max_tokens": 100}`,
		`{"temperature": 0.5, "max_tokens": 750, "top_p": 0.95}`,
		`{"frequency_penalty": 0.1, "presence_penalty": 0.1}`,
		`{"temperature": 0.8, "stop": [";", "\n\n"]}`,
	}
}

// TestScenario represents a comprehensive test scenario
type TestScenario struct {
	Name         string
	Description  string
	Prompts      []string
	Configs      []string
	ExpectedKeywords []string
	ShouldSucceed bool
	MaxDuration  time.Duration
	Category     string
}

// GetTestScenarios returns predefined test scenarios
func GetTestScenarios() []TestScenario {
	return []TestScenario{
		{
			Name:        "BasicSQLGeneration",
			Description: "Test basic SQL statement generation",
			Prompts: []string{
				"CREATE TABLE users (id INT, name VARCHAR(100))",
				"SELECT * FROM users",
				"INSERT INTO users (name) VALUES ('test')",
			},
			Configs:          []string{`{"temperature": 0.3}`},
			ExpectedKeywords: []string{"CREATE", "SELECT", "INSERT"},
			ShouldSucceed:    true,
			MaxDuration:      10 * time.Second,
			Category:        "basic",
		},
		{
			Name:        "ComplexQueryGeneration",
			Description: "Test complex SQL query generation",
			Prompts: []string{
				"Create a query to find users with more than 10 orders in the last month",
				"Generate SQL for a report showing monthly revenue by product category",
			},
			Configs:          []string{`{"temperature": 0.7, "max_tokens": 1000}`},
			ExpectedKeywords: []string{"SELECT", "FROM", "WHERE", "GROUP BY"},
			ShouldSucceed:    true,
			MaxDuration:      15 * time.Second,
			Category:        "complex",
		},
		{
			Name:        "SchemaDesign",
			Description: "Test database schema design generation",
			Prompts: []string{
				"Design a database schema for an e-commerce platform",
				"Create tables for a blog system with users, posts, and comments",
			},
			Configs:          []string{`{"temperature": 0.5, "max_tokens": 1500}`},
			ExpectedKeywords: []string{"CREATE TABLE", "PRIMARY KEY", "FOREIGN KEY"},
			ShouldSucceed:    true,
			MaxDuration:      20 * time.Second,
			Category:        "schema",
		},
		{
			Name:        "ErrorHandling",
			Description: "Test error handling for invalid inputs",
			Prompts: []string{
				"", // Empty prompt
				"   ", // Whitespace only
			},
			Configs:       []string{`{}`},
			ShouldSucceed: false,
			MaxDuration:   5 * time.Second,
			Category:     "error",
		},
		{
			Name:        "PerformanceTest",
			Description: "Test performance with various prompt sizes",
			Prompts: []string{
				"SELECT 1", // Small prompt
				generateLargePrompt(), // Large prompt
			},
			Configs:       []string{`{"temperature": 0.5}`},
			ShouldSucceed: true,
			MaxDuration:   30 * time.Second,
			Category:     "performance",
		},
	}
}

// generateLargePrompt creates a large prompt for performance testing
func generateLargePrompt() string {
	base := "Create a comprehensive database schema including tables for: "
	tables := []string{
		"users with authentication and profile information",
		"products with categories, inventory, and pricing",
		"orders with line items and shipping details",
		"payments with transaction history",
		"reviews and ratings",
		"shopping carts and wishlists",
		"notifications and messaging",
		"audit logs and system events",
		"file attachments and media",
		"configuration and settings",
	}

	prompt := base
	for i, table := range tables {
		prompt += fmt.Sprintf("%d) %s, ", i+1, table)
	}
	prompt += "with proper relationships, indexes, and constraints for optimal performance."

	return prompt
}

// ValidationRules defines validation rules for test responses
type ValidationRules struct {
	RequiredFields    []string
	ForbiddenFields   []string
	RequiredKeywords  []string
	ForbiddenKeywords []string
	MinContentLength  int
	MaxContentLength  int
	ValidateJSON      []string // Fields that should be valid JSON
}

// GetValidationRules returns validation rules for different test categories
func GetValidationRules() map[string]ValidationRules {
	return map[string]ValidationRules{
		"capabilities": {
			RequiredFields:   []string{"success", "capabilities", "description", "version"},
			RequiredKeywords: []string{"AI", "plugin"},
			ValidateJSON:     []string{"capabilities"},
		},
		"generation_success": {
			RequiredFields:   []string{"success", "content"},
			RequiredKeywords: []string{},
			MinContentLength: 10,
			MaxContentLength: 10000,
		},
		"generation_error": {
			RequiredFields:    []string{"success", "error"},
			ForbiddenFields:   []string{"content"},
			RequiredKeywords:  []string{},
		},
		"sql_content": {
			RequiredKeywords: []string{}, // Will be set based on prompt type
			ForbiddenKeywords: []string{"undefined", "null", "error"},
		},
	}
}

// Performance thresholds and limits
const (
	MaxStartupTime        = 30 * time.Second
	MaxCapabilitiesTime   = 5 * time.Second
	MaxSimpleGeneration   = 10 * time.Second
	MaxComplexGeneration  = 20 * time.Second
	MaxConcurrentRequests = 10
	MinSuccessRate        = 0.95 // 95% success rate expected
)

// Environment configuration keys
const (
	EnvPluginBinary = "ATEST_EXT_AI_BINARY"
	EnvPluginConfig = "ATEST_EXT_AI_CONFIG"
	EnvPluginSocket = "ATEST_EXT_AI_SOCKET"
	EnvLogLevel     = "LOG_LEVEL"
	EnvTestTimeout  = "TEST_TIMEOUT"
	EnvSkipSlow     = "SKIP_SLOW_TESTS"
)

// Helper functions for test setup
func (tc *TestConfig) Validate() error {
	if tc.PluginBinaryPath == "" {
		return fmt.Errorf("plugin binary path is required")
	}

	if _, err := os.Stat(tc.PluginBinaryPath); err != nil {
		return fmt.Errorf("plugin binary not found at %s: %v", tc.PluginBinaryPath, err)
	}

	if tc.ConfigPath != "" {
		if _, err := os.Stat(tc.ConfigPath); err != nil {
			return fmt.Errorf("config file not found at %s: %v", tc.ConfigPath, err)
		}
	}

	if tc.SocketPath == "" {
		tc.SocketPath = generateSocketPath()
	}

	return nil
}

func (tc *TestConfig) GetEnvironment() []string {
	env := []string{
		"AI_PLUGIN_SOCKET_PATH=" + tc.SocketPath,
		"LOG_LEVEL=" + tc.LogLevel,
	}

	if tc.ConfigPath != "" {
		env = append(env, "CONFIG_PATH="+tc.ConfigPath)
	}

	return env
}

func (tc *TestConfig) Cleanup() {
	if tc.SocketPath != "" {
		os.Remove(tc.SocketPath)
	}
}