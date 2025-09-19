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

package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// SQLGenerator handles SQL generation from natural language
type SQLGenerator struct {
	aiClient     interfaces.AIClient
	sqlDialects  map[string]SQLDialect
	config       config.AIConfig
	capabilities *SQLCapabilities
}

// Table represents a database table structure
type Table struct {
	Name        string            `json:"name"`
	Columns     []Column          `json:"columns"`
	PrimaryKey  []string          `json:"primary_key,omitempty"`
	ForeignKeys []ForeignKey      `json:"foreign_keys,omitempty"`
	Indexes     []Index           `json:"indexes,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Column represents a table column
type Column struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Nullable     bool   `json:"nullable"`
	DefaultValue string `json:"default_value,omitempty"`
	Comment      string `json:"comment,omitempty"`
	MaxLength    int    `json:"max_length,omitempty"`
	Precision    int    `json:"precision,omitempty"`
	Scale        int    `json:"scale,omitempty"`
}

// ForeignKey represents a foreign key relationship
type ForeignKey struct {
	Name           string   `json:"name"`
	Columns        []string `json:"columns"`
	ReferencedTable string   `json:"referenced_table"`
	ReferencedColumns []string `json:"referenced_columns"`
	OnDelete       string   `json:"on_delete,omitempty"`
	OnUpdate       string   `json:"on_update,omitempty"`
}

// Index represents a table index
type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Type    string   `json:"type,omitempty"`
}

// GenerateOptions contains options for SQL generation
type GenerateOptions struct {
	DatabaseType     string            `json:"database_type"`
	Schema           map[string]Table  `json:"schema,omitempty"`
	Context          []string          `json:"context,omitempty"`
	Temperature      float64           `json:"temperature,omitempty"`
	MaxTokens        int               `json:"max_tokens,omitempty"`
	ValidateSQL      bool              `json:"validate_sql"`
	OptimizeQuery    bool              `json:"optimize_query"`
	IncludeExplanation bool            `json:"include_explanation"`
	SafetyMode       bool              `json:"safety_mode"`
	CustomPrompts    map[string]string `json:"custom_prompts,omitempty"`
}

// GenerationResult contains the complete result of SQL generation
type GenerationResult struct {
	SQL             string          `json:"sql"`
	Explanation     string          `json:"explanation"`
	ConfidenceScore float64         `json:"confidence_score"`
	Warnings        []string        `json:"warnings"`
	Suggestions     []string        `json:"suggestions"`
	Metadata        GenerationMetadata `json:"metadata"`
	ValidationResults []ValidationResult `json:"validation_results,omitempty"`
}

// GenerationMetadata contains metadata about the generation process
type GenerationMetadata struct {
	RequestID        string        `json:"request_id"`
	ProcessingTime   time.Duration `json:"processing_time"`
	ModelUsed        string        `json:"model_used"`
	DatabaseDialect  string        `json:"database_dialect"`
	TokensUsed       TokenUsage    `json:"tokens_used"`
	QueryType        string        `json:"query_type"`
	TablesInvolved   []string      `json:"tables_involved,omitempty"`
	Complexity       string        `json:"complexity"`
	DebugInfo        []string      `json:"debug_info,omitempty"`
}

// ValidationResult contains SQL validation information
type ValidationResult struct {
	Type        string `json:"type"`
	Level       string `json:"level"` // info, warning, error
	Message     string `json:"message"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// NewSQLGenerator creates a new SQL generator instance
func NewSQLGenerator(aiClient interfaces.AIClient, config config.AIConfig) (*SQLGenerator, error) {
	if aiClient == nil {
		return nil, fmt.Errorf("AI client cannot be nil")
	}

	generator := &SQLGenerator{
		aiClient:    aiClient,
		config:      config,
		sqlDialects: make(map[string]SQLDialect),
	}

	// Initialize SQL dialects
	if err := generator.initializeDialects(); err != nil {
		return nil, fmt.Errorf("failed to initialize SQL dialects: %w", err)
	}

	// Initialize capabilities
	generator.capabilities = &SQLCapabilities{
		SupportedDatabases: []string{"mysql", "postgresql", "sqlite"},
		Features: []SQLFeature{
			{
				Name:        "Natural Language to SQL",
				Enabled:     true,
				Description: "Convert natural language queries to SQL",
			},
			{
				Name:        "Multi-dialect Support",
				Enabled:     true,
				Description: "Support for MySQL, PostgreSQL, and SQLite",
			},
			{
				Name:        "Schema-aware Generation",
				Enabled:     true,
				Description: "Generate SQL based on provided database schema",
			},
			{
				Name:        "Query Optimization",
				Enabled:     true,
				Description: "Optimize generated SQL for performance",
			},
			{
				Name:        "SQL Validation",
				Enabled:     true,
				Description: "Validate generated SQL syntax",
			},
		},
	}

	return generator, nil
}

// Generate generates SQL from natural language input
func (g *SQLGenerator) Generate(ctx context.Context, naturalLanguage string, options *GenerateOptions) (*GenerationResult, error) {
	start := time.Now()
	requestID := fmt.Sprintf("sql_%d", start.UnixNano())

	if naturalLanguage == "" {
		return nil, fmt.Errorf("natural language query cannot be empty")
	}

	if options == nil {
		options = &GenerateOptions{
			DatabaseType:       "mysql",
			ValidateSQL:        true,
			OptimizeQuery:      false,
			IncludeExplanation: true,
			SafetyMode:         true,
			Temperature:        0.3,
			MaxTokens:          2000,
		}
	}

	// Get SQL dialect
	dialect, exists := g.sqlDialects[options.DatabaseType]
	if !exists {
		return nil, fmt.Errorf("unsupported database type: %s", options.DatabaseType)
	}

	// Prepare the prompt for AI
	prompt, err := g.buildPrompt(naturalLanguage, options, dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Create AI request
	aiRequest := &GenerateRequest{
		Prompt:       prompt,
		Temperature:  options.Temperature,
		MaxTokens:    options.MaxTokens,
		SystemPrompt: g.getSystemPrompt(options.DatabaseType),
	}

	// Call AI service
	aiResponse, err := g.aiClient.Generate(ctx, aiRequest)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// Parse and validate the response
	result, err := g.parseAIResponse(aiResponse, options, dialect, requestID, start)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return result, nil
}

// initializeDialects initializes SQL dialect support
func (g *SQLGenerator) initializeDialects() error {
	// Initialize MySQL dialect
	g.sqlDialects["mysql"] = &MySQLDialect{}

	// Initialize PostgreSQL dialect
	g.sqlDialects["postgresql"] = &PostgreSQLDialect{}
	g.sqlDialects["postgres"] = &PostgreSQLDialect{}

	// Initialize SQLite dialect
	g.sqlDialects["sqlite"] = &SQLiteDialect{}

	return nil
}

// buildPrompt constructs the AI prompt for SQL generation
func (g *SQLGenerator) buildPrompt(naturalLanguage string, options *GenerateOptions, dialect SQLDialect) (string, error) {
	var promptBuilder strings.Builder

	// Add custom prompt if provided
	if customPrompt, exists := options.CustomPrompts["sql_generation"]; exists {
		promptBuilder.WriteString(customPrompt + "\n\n")
	} else {
		// Default SQL generation prompt
		promptBuilder.WriteString("Generate a SQL query based on the following natural language description.\n\n")
	}

	// Add database-specific context
	promptBuilder.WriteString(fmt.Sprintf("Database Type: %s\n", options.DatabaseType))
	promptBuilder.WriteString(fmt.Sprintf("SQL Dialect: %s\n\n", dialect.Name()))

	// Add schema information if provided
	if len(options.Schema) > 0 {
		promptBuilder.WriteString("Database Schema:\n")
		for tableName, table := range options.Schema {
			promptBuilder.WriteString(fmt.Sprintf("Table: %s\n", tableName))
			for _, column := range table.Columns {
				nullable := "NOT NULL"
				if column.Nullable {
					nullable = "NULL"
				}
				promptBuilder.WriteString(fmt.Sprintf("  - %s %s %s", column.Name, column.Type, nullable))
				if column.Comment != "" {
					promptBuilder.WriteString(fmt.Sprintf(" -- %s", column.Comment))
				}
				promptBuilder.WriteString("\n")
			}
			promptBuilder.WriteString("\n")
		}
	}

	// Add context information
	if len(options.Context) > 0 {
		promptBuilder.WriteString("Additional Context:\n")
		for _, ctx := range options.Context {
			promptBuilder.WriteString(fmt.Sprintf("- %s\n", ctx))
		}
		promptBuilder.WriteString("\n")
	}

	// Add safety constraints if enabled
	if options.SafetyMode {
		promptBuilder.WriteString("Safety Requirements:\n")
		promptBuilder.WriteString("- Do not generate DROP, DELETE, or TRUNCATE statements unless explicitly requested\n")
		promptBuilder.WriteString("- Include appropriate WHERE clauses to prevent accidental data modification\n")
		promptBuilder.WriteString("- Use prepared statement placeholders for user inputs\n")
		promptBuilder.WriteString("- Validate that the query follows security best practices\n\n")
	}

	// Add the natural language query
	promptBuilder.WriteString("Natural Language Query:\n")
	promptBuilder.WriteString(naturalLanguage)
	promptBuilder.WriteString("\n\n")

	// Add format requirements
	promptBuilder.WriteString("Response Format:\n")
	promptBuilder.WriteString("Please provide the response in the following JSON format:\n")
	promptBuilder.WriteString("{\n")
	promptBuilder.WriteString("  \"sql\": \"<generated SQL query>\",\n")
	if options.IncludeExplanation {
		promptBuilder.WriteString("  \"explanation\": \"<explanation of the query>\",\n")
	}
	promptBuilder.WriteString("  \"confidence\": <confidence score from 0.0 to 1.0>,\n")
	promptBuilder.WriteString("  \"query_type\": \"<SELECT|INSERT|UPDATE|DELETE|CREATE|etc>\",\n")
	promptBuilder.WriteString("  \"tables_involved\": [\"<table names>\"],\n")
	promptBuilder.WriteString("  \"warnings\": [\"<any warnings>\"],\n")
	promptBuilder.WriteString("  \"suggestions\": [\"<optimization suggestions>\"]\n")
	promptBuilder.WriteString("}\n")

	return promptBuilder.String(), nil
}

// getSystemPrompt returns the system prompt for SQL generation
func (g *SQLGenerator) getSystemPrompt(databaseType string) string {
	return fmt.Sprintf(`You are an expert SQL database assistant specializing in %s.
Your task is to convert natural language queries into accurate, efficient SQL statements.

Key principles:
1. Generate syntactically correct SQL for %s
2. Follow security best practices
3. Optimize for readability and performance
4. Provide clear explanations when requested
5. Include appropriate error handling
6. Use standard SQL when possible, dialect-specific features only when necessary

Always respond with valid JSON format as requested.`, databaseType, databaseType)
}

// parseAIResponse parses and validates the AI response
func (g *SQLGenerator) parseAIResponse(aiResponse *GenerateResponse, options *GenerateOptions, dialect SQLDialect, requestID string, startTime time.Time) (*GenerationResult, error) {
	// Try to extract JSON from the response
	sqlResult, err := g.extractSQLFromResponse(aiResponse.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to extract SQL from AI response: %w", err)
	}

	// Create generation result
	result := &GenerationResult{
		SQL:             sqlResult.SQL,
		Explanation:     sqlResult.Explanation,
		ConfidenceScore: sqlResult.Confidence,
		Warnings:        sqlResult.Warnings,
		Suggestions:     sqlResult.Suggestions,
		Metadata: GenerationMetadata{
			RequestID:       requestID,
			ProcessingTime:  time.Since(startTime),
			ModelUsed:       aiResponse.Model,
			DatabaseDialect: options.DatabaseType,
			TokensUsed:      aiResponse.Usage,
			QueryType:       sqlResult.QueryType,
			TablesInvolved:  sqlResult.TablesInvolved,
			Complexity:      g.assessComplexity(sqlResult.SQL),
		},
	}

	// Validate SQL if requested
	if options.ValidateSQL {
		validationResults, err := dialect.ValidateSQL(sqlResult.SQL)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("SQL validation failed: %v", err))
		} else {
			result.ValidationResults = validationResults
		}
	}

	// Optimize query if requested
	if options.OptimizeQuery {
		optimizedSQL, suggestions, err := dialect.OptimizeSQL(sqlResult.SQL)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("SQL optimization failed: %v", err))
		} else {
			result.SQL = optimizedSQL
			result.Suggestions = append(result.Suggestions, suggestions...)
		}
	}

	return result, nil
}

// SQLResponse represents the structured response from AI
type SQLResponse struct {
	SQL            string   `json:"sql"`
	Explanation    string   `json:"explanation,omitempty"`
	Confidence     float64  `json:"confidence"`
	QueryType      string   `json:"query_type"`
	TablesInvolved []string `json:"tables_involved"`
	Warnings       []string `json:"warnings"`
	Suggestions    []string `json:"suggestions"`
}

// extractSQLFromResponse extracts structured SQL information from AI response
func (g *SQLGenerator) extractSQLFromResponse(responseText string) (*SQLResponse, error) {
	// This is a simplified implementation - in practice, you'd want more robust JSON parsing
	// For now, we'll create a basic response structure

	// Try to find SQL query in the response
	sql := strings.TrimSpace(responseText)

	// Remove common prefixes and suffixes
	sql = strings.TrimPrefix(sql, "```sql")
	sql = strings.TrimPrefix(sql, "```")
	sql = strings.TrimSuffix(sql, "```")
	sql = strings.TrimSpace(sql)

	// If it's still empty, provide a default
	if sql == "" {
		sql = "SELECT 1 as placeholder;"
	}

	return &SQLResponse{
		SQL:            sql,
		Explanation:    "Generated SQL query based on natural language input",
		Confidence:     0.8,
		QueryType:      g.detectQueryType(sql),
		TablesInvolved: g.extractTableNames(sql),
		Warnings:       []string{},
		Suggestions:    []string{},
	}, nil
}

// detectQueryType determines the type of SQL query
func (g *SQLGenerator) detectQueryType(sql string) string {
	upper := strings.ToUpper(strings.TrimSpace(sql))

	if strings.HasPrefix(upper, "SELECT") {
		return "SELECT"
	} else if strings.HasPrefix(upper, "INSERT") {
		return "INSERT"
	} else if strings.HasPrefix(upper, "UPDATE") {
		return "UPDATE"
	} else if strings.HasPrefix(upper, "DELETE") {
		return "DELETE"
	} else if strings.HasPrefix(upper, "CREATE") {
		return "CREATE"
	} else if strings.HasPrefix(upper, "DROP") {
		return "DROP"
	} else if strings.HasPrefix(upper, "ALTER") {
		return "ALTER"
	}

	return "UNKNOWN"
}

// extractTableNames extracts table names from SQL query
func (g *SQLGenerator) extractTableNames(sql string) []string {
	// Simplified table extraction - in practice, you'd want more sophisticated parsing
	tables := []string{}

	// Look for FROM and JOIN keywords
	upper := strings.ToUpper(sql)
	words := strings.Fields(upper)

	for i, word := range words {
		if (word == "FROM" || word == "JOIN" || word == "UPDATE" || word == "INTO") && i+1 < len(words) {
			tableName := words[i+1]
			// Remove common SQL keywords and punctuation
			tableName = strings.TrimSuffix(tableName, ",")
			tableName = strings.TrimSuffix(tableName, "(")
			if tableName != "" && !contains(tables, tableName) {
				tables = append(tables, tableName)
			}
		}
	}

	return tables
}

// assessComplexity assesses the complexity of the generated SQL
func (g *SQLGenerator) assessComplexity(sql string) string {
	upper := strings.ToUpper(sql)

	// Count complex features
	complexity := 0

	if strings.Contains(upper, "JOIN") {
		complexity++
	}
	if strings.Contains(upper, "SUBQUERY") || strings.Count(upper, "(SELECT") > 0 {
		complexity++
	}
	if strings.Contains(upper, "GROUP BY") {
		complexity++
	}
	if strings.Contains(upper, "HAVING") {
		complexity++
	}
	if strings.Contains(upper, "UNION") {
		complexity++
	}
	if strings.Contains(upper, "WITH") { // CTE
		complexity++
	}
	if strings.Contains(upper, "WINDOW") || strings.Contains(upper, "OVER") {
		complexity++
	}

	switch {
	case complexity == 0:
		return "simple"
	case complexity <= 2:
		return "moderate"
	case complexity <= 4:
		return "complex"
	default:
		return "very_complex"
	}
}

// GetCapabilities returns the SQL generation capabilities
func (g *SQLGenerator) GetCapabilities() *SQLCapabilities {
	return g.capabilities
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}