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
	"fmt"
	"regexp"
	"strings"
)

// SQLDialect defines the interface for database-specific SQL handling
type SQLDialect interface {
	// Name returns the name of the SQL dialect
	Name() string

	// ValidateSQL validates SQL syntax for this dialect
	ValidateSQL(sql string) ([]ValidationResult, error)

	// OptimizeSQL optimizes SQL query for this dialect
	OptimizeSQL(sql string) (string, []string, error)

	// FormatSQL formats SQL query according to dialect conventions
	FormatSQL(sql string) (string, error)

	// GetDataTypes returns supported data types for this dialect
	GetDataTypes() []DataType

	// GetFunctions returns supported functions for this dialect
	GetFunctions() []Function

	// GetKeywords returns reserved keywords for this dialect
	GetKeywords() []string

	// TransformSQL transforms SQL from one dialect to another
	TransformSQL(sql string, targetDialect string) (string, error)
}

// DataType represents a database data type
type DataType struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"` // numeric, string, date, boolean, etc.
	Aliases     []string `json:"aliases,omitempty"`
	MaxLength   int      `json:"max_length,omitempty"`
	DefaultSize int      `json:"default_size,omitempty"`
	Precision   int      `json:"precision,omitempty"`
	Scale       int      `json:"scale,omitempty"`
}

// Function represents a database function
type Function struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"` // aggregate, string, date, math, etc.
	Description string   `json:"description"`
	Syntax      string   `json:"syntax"`
	Examples    []string `json:"examples,omitempty"`
}

// MySQLDialect implements SQLDialect for MySQL
type MySQLDialect struct{}

func (d *MySQLDialect) Name() string {
	return "MySQL"
}

func (d *MySQLDialect) ValidateSQL(sql string) ([]ValidationResult, error) {
	var results []ValidationResult

	// Basic syntax validation
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return []ValidationResult{{
			Type:    "syntax",
			Level:   "error",
			Message: "Empty SQL statement",
		}}, nil
	}

	// Check for common MySQL syntax issues
	upper := strings.ToUpper(sql)

	// Check for proper statement termination
	if !strings.HasSuffix(strings.TrimSpace(sql), ";") {
		results = append(results, ValidationResult{
			Type:       "syntax",
			Level:      "warning",
			Message:    "SQL statement should end with semicolon",
			Suggestion: "Add ';' at the end of the statement",
		})
	}

	// Check for MySQL-specific issues
	if strings.Contains(upper, "LIMIT") && !regexp.MustCompile(`LIMIT\s+\d+(\s*,\s*\d+)?`).MatchString(upper) {
		results = append(results, ValidationResult{
			Type:    "syntax",
			Level:   "error",
			Message: "Invalid LIMIT syntax for MySQL",
		})
	}

	// Check for reserved keywords used as identifiers (not as SQL commands)
	// We'll only check for keywords that might be used as table or column names
	problematicKeywords := []string{"ORDER", "GROUP", "KEY", "INDEX", "TABLE", "DATABASE"}
	for _, keyword := range problematicKeywords {
		if strings.Contains(upper, keyword+" ") && !strings.Contains(upper, "`"+keyword+"`") {
			// More sophisticated check to see if it's used as identifier
			if !isKeywordUsedAsCommand(upper, keyword) {
				results = append(results, ValidationResult{
					Type:       "naming",
					Level:      "warning",
					Message:    fmt.Sprintf("'%s' might be a reserved keyword in MySQL", keyword),
					Suggestion: fmt.Sprintf("Use backticks if using as identifier: `%s`", keyword),
				})
			}
		}
	}

	return results, nil
}

func (d *MySQLDialect) OptimizeSQL(sql string) (string, []string, error) {
	var suggestions []string
	optimizedSQL := sql

	upper := strings.ToUpper(sql)

	// Suggest using LIMIT for potentially large result sets
	if strings.Contains(upper, "SELECT") && !strings.Contains(upper, "LIMIT") && !strings.Contains(upper, "WHERE") {
		suggestions = append(suggestions, "Consider adding a LIMIT clause to prevent large result sets")
	}

	// Suggest using indexes for WHERE clauses
	if strings.Contains(upper, "WHERE") {
		suggestions = append(suggestions, "Ensure appropriate indexes exist for WHERE clause columns")
	}

	// Suggest using EXISTS instead of IN for subqueries
	if strings.Contains(upper, "IN (SELECT") {
		suggestions = append(suggestions, "Consider using EXISTS instead of IN with subqueries for better performance")
	}

	return optimizedSQL, suggestions, nil
}

func (d *MySQLDialect) FormatSQL(sql string) (string, error) {
	// Basic SQL formatting - indent and add line breaks
	formatted := strings.TrimSpace(sql)

	// Add line breaks after major keywords
	keywords := []string{"SELECT", "FROM", "WHERE", "GROUP BY", "HAVING", "ORDER BY", "LIMIT"}
	for _, keyword := range keywords {
		pattern := regexp.MustCompile(`(?i)\b` + keyword + `\b`)
		formatted = pattern.ReplaceAllString(formatted, "\n"+keyword)
	}

	return strings.TrimSpace(formatted), nil
}

func (d *MySQLDialect) GetDataTypes() []DataType {
	return []DataType{
		{Name: "INT", Category: "numeric", Aliases: []string{"INTEGER"}},
		{Name: "BIGINT", Category: "numeric"},
		{Name: "DECIMAL", Category: "numeric", Precision: 65, Scale: 30},
		{Name: "FLOAT", Category: "numeric"},
		{Name: "DOUBLE", Category: "numeric"},
		{Name: "VARCHAR", Category: "string", MaxLength: 65535},
		{Name: "CHAR", Category: "string", MaxLength: 255},
		{Name: "TEXT", Category: "string"},
		{Name: "LONGTEXT", Category: "string"},
		{Name: "DATE", Category: "date"},
		{Name: "DATETIME", Category: "date"},
		{Name: "TIMESTAMP", Category: "date"},
		{Name: "BOOLEAN", Category: "boolean", Aliases: []string{"BOOL"}},
		{Name: "JSON", Category: "json"},
		{Name: "BLOB", Category: "binary"},
	}
}

func (d *MySQLDialect) GetFunctions() []Function {
	return []Function{
		{Name: "COUNT", Category: "aggregate", Description: "Count rows", Syntax: "COUNT(column)", Examples: []string{"COUNT(*)", "COUNT(id)"}},
		{Name: "SUM", Category: "aggregate", Description: "Sum values", Syntax: "SUM(column)", Examples: []string{"SUM(amount)"}},
		{Name: "AVG", Category: "aggregate", Description: "Average values", Syntax: "AVG(column)", Examples: []string{"AVG(price)"}},
		{Name: "MAX", Category: "aggregate", Description: "Maximum value", Syntax: "MAX(column)", Examples: []string{"MAX(created_at)"}},
		{Name: "MIN", Category: "aggregate", Description: "Minimum value", Syntax: "MIN(column)", Examples: []string{"MIN(price)"}},
		{Name: "CONCAT", Category: "string", Description: "Concatenate strings", Syntax: "CONCAT(str1, str2, ...)", Examples: []string{"CONCAT(first_name, ' ', last_name)"}},
		{Name: "LENGTH", Category: "string", Description: "String length", Syntax: "LENGTH(str)", Examples: []string{"LENGTH(description)"}},
		{Name: "SUBSTRING", Category: "string", Description: "Extract substring", Syntax: "SUBSTRING(str, pos, len)", Examples: []string{"SUBSTRING(name, 1, 10)"}},
		{Name: "NOW", Category: "date", Description: "Current timestamp", Syntax: "NOW()", Examples: []string{"NOW()"}},
		{Name: "DATE", Category: "date", Description: "Extract date part", Syntax: "DATE(datetime)", Examples: []string{"DATE(created_at)"}},
	}
}

func (d *MySQLDialect) GetKeywords() []string {
	return []string{
		"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER",
		"TABLE", "INDEX", "DATABASE", "SCHEMA", "VIEW", "PROCEDURE", "FUNCTION", "TRIGGER",
		"PRIMARY", "FOREIGN", "KEY", "UNIQUE", "NOT", "NULL", "DEFAULT", "AUTO_INCREMENT",
		"AND", "OR", "IN", "LIKE", "BETWEEN", "EXISTS", "IS", "CASE", "WHEN", "THEN", "ELSE",
		"GROUP", "BY", "ORDER", "HAVING", "LIMIT", "OFFSET", "UNION", "JOIN", "LEFT", "RIGHT",
		"INNER", "OUTER", "ON", "AS", "DISTINCT", "ALL", "ASC", "DESC",
	}
}

func (d *MySQLDialect) TransformSQL(sql string, targetDialect string) (string, error) {
	switch targetDialect {
	case "postgresql":
		return d.transformToPostgreSQL(sql)
	case "sqlite":
		return d.transformToSQLite(sql)
	default:
		return sql, fmt.Errorf("unsupported target dialect: %s", targetDialect)
	}
}

func (d *MySQLDialect) transformToPostgreSQL(sql string) (string, error) {
	// Transform MySQL-specific syntax to PostgreSQL
	transformed := sql

	// Replace backticks with double quotes
	transformed = strings.ReplaceAll(transformed, "`", "\"")

	// Replace LIMIT x, y with LIMIT y OFFSET x
	limitPattern := regexp.MustCompile(`(?i)LIMIT\s+(\d+)\s*,\s*(\d+)`)
	transformed = limitPattern.ReplaceAllString(transformed, "LIMIT $2 OFFSET $1")

	// Replace AUTO_INCREMENT with SERIAL
	transformed = strings.ReplaceAll(strings.ToUpper(transformed), "AUTO_INCREMENT", "SERIAL")

	return transformed, nil
}

func (d *MySQLDialect) transformToSQLite(sql string) (string, error) {
	// Transform MySQL-specific syntax to SQLite
	transformed := sql

	// Remove backticks
	transformed = strings.ReplaceAll(transformed, "`", "")

	// Replace some MySQL functions with SQLite equivalents
	transformed = strings.ReplaceAll(strings.ToUpper(transformed), "NOW()", "DATETIME('now')")

	return transformed, nil
}

// PostgreSQLDialect implements SQLDialect for PostgreSQL
type PostgreSQLDialect struct{}

func (d *PostgreSQLDialect) Name() string {
	return "PostgreSQL"
}

func (d *PostgreSQLDialect) ValidateSQL(sql string) ([]ValidationResult, error) {
	var results []ValidationResult

	sql = strings.TrimSpace(sql)
	if sql == "" {
		return []ValidationResult{{
			Type:    "syntax",
			Level:   "error",
			Message: "Empty SQL statement",
		}}, nil
	}

	upper := strings.ToUpper(sql)

	// Check for proper statement termination
	if !strings.HasSuffix(strings.TrimSpace(sql), ";") {
		results = append(results, ValidationResult{
			Type:       "syntax",
			Level:      "warning",
			Message:    "SQL statement should end with semicolon",
			Suggestion: "Add ';' at the end of the statement",
		})
	}

	// Check for PostgreSQL-specific issues
	if strings.Contains(upper, "LIMIT") && strings.Contains(upper, ",") {
		results = append(results, ValidationResult{
			Type:       "syntax",
			Level:      "error",
			Message:    "PostgreSQL uses LIMIT x OFFSET y syntax, not LIMIT x, y",
			Suggestion: "Use LIMIT count OFFSET start format",
		})
	}

	// Check for identifier quoting
	if strings.Contains(sql, "`") {
		results = append(results, ValidationResult{
			Type:       "syntax",
			Level:      "warning",
			Message:    "PostgreSQL uses double quotes for identifiers, not backticks",
			Suggestion: "Use double quotes (\") instead of backticks (`)",
		})
	}

	return results, nil
}

func (d *PostgreSQLDialect) OptimizeSQL(sql string) (string, []string, error) {
	var suggestions []string
	optimizedSQL := sql

	upper := strings.ToUpper(sql)

	// Suggest using LIMIT for potentially large result sets
	if strings.Contains(upper, "SELECT") && !strings.Contains(upper, "LIMIT") {
		suggestions = append(suggestions, "Consider adding a LIMIT clause to prevent large result sets")
	}

	// Suggest using indexes
	if strings.Contains(upper, "WHERE") {
		suggestions = append(suggestions, "Ensure appropriate indexes exist for WHERE clause columns")
	}

	// Suggest using EXISTS instead of IN for subqueries
	if strings.Contains(upper, "IN (SELECT") {
		suggestions = append(suggestions, "Consider using EXISTS instead of IN with subqueries for better performance")
	}

	return optimizedSQL, suggestions, nil
}

func (d *PostgreSQLDialect) FormatSQL(sql string) (string, error) {
	// Basic SQL formatting
	formatted := strings.TrimSpace(sql)

	keywords := []string{"SELECT", "FROM", "WHERE", "GROUP BY", "HAVING", "ORDER BY", "LIMIT", "OFFSET"}
	for _, keyword := range keywords {
		pattern := regexp.MustCompile(`(?i)\b` + keyword + `\b`)
		formatted = pattern.ReplaceAllString(formatted, "\n"+keyword)
	}

	return strings.TrimSpace(formatted), nil
}

func (d *PostgreSQLDialect) GetDataTypes() []DataType {
	return []DataType{
		{Name: "INTEGER", Category: "numeric", Aliases: []string{"INT", "INT4"}},
		{Name: "BIGINT", Category: "numeric", Aliases: []string{"INT8"}},
		{Name: "DECIMAL", Category: "numeric", Aliases: []string{"NUMERIC"}},
		{Name: "REAL", Category: "numeric", Aliases: []string{"FLOAT4"}},
		{Name: "DOUBLE PRECISION", Category: "numeric", Aliases: []string{"FLOAT8"}},
		{Name: "VARCHAR", Category: "string", Aliases: []string{"CHARACTER VARYING"}},
		{Name: "CHAR", Category: "string", Aliases: []string{"CHARACTER"}},
		{Name: "TEXT", Category: "string"},
		{Name: "DATE", Category: "date"},
		{Name: "TIMESTAMP", Category: "date"},
		{Name: "TIMESTAMPTZ", Category: "date", Aliases: []string{"TIMESTAMP WITH TIME ZONE"}},
		{Name: "BOOLEAN", Category: "boolean", Aliases: []string{"BOOL"}},
		{Name: "JSON", Category: "json"},
		{Name: "JSONB", Category: "json"},
		{Name: "UUID", Category: "uuid"},
		{Name: "SERIAL", Category: "numeric"},
		{Name: "BIGSERIAL", Category: "numeric"},
	}
}

func (d *PostgreSQLDialect) GetFunctions() []Function {
	return []Function{
		{Name: "COUNT", Category: "aggregate", Description: "Count rows", Syntax: "COUNT(column)", Examples: []string{"COUNT(*)", "COUNT(id)"}},
		{Name: "SUM", Category: "aggregate", Description: "Sum values", Syntax: "SUM(column)", Examples: []string{"SUM(amount)"}},
		{Name: "AVG", Category: "aggregate", Description: "Average values", Syntax: "AVG(column)", Examples: []string{"AVG(price)"}},
		{Name: "MAX", Category: "aggregate", Description: "Maximum value", Syntax: "MAX(column)", Examples: []string{"MAX(created_at)"}},
		{Name: "MIN", Category: "aggregate", Description: "Minimum value", Syntax: "MIN(column)", Examples: []string{"MIN(price)"}},
		{Name: "CONCAT", Category: "string", Description: "Concatenate strings", Syntax: "CONCAT(str1, str2, ...)", Examples: []string{"CONCAT(first_name, ' ', last_name)"}},
		{Name: "LENGTH", Category: "string", Description: "String length", Syntax: "LENGTH(str)", Examples: []string{"LENGTH(description)"}},
		{Name: "SUBSTRING", Category: "string", Description: "Extract substring", Syntax: "SUBSTRING(str FROM pos FOR len)", Examples: []string{"SUBSTRING(name FROM 1 FOR 10)"}},
		{Name: "NOW", Category: "date", Description: "Current timestamp", Syntax: "NOW()", Examples: []string{"NOW()"}},
		{Name: "CURRENT_DATE", Category: "date", Description: "Current date", Syntax: "CURRENT_DATE", Examples: []string{"CURRENT_DATE"}},
		{Name: "EXTRACT", Category: "date", Description: "Extract date part", Syntax: "EXTRACT(field FROM source)", Examples: []string{"EXTRACT(YEAR FROM created_at)"}},
	}
}

func (d *PostgreSQLDialect) GetKeywords() []string {
	return []string{
		"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER",
		"TABLE", "INDEX", "DATABASE", "SCHEMA", "VIEW", "PROCEDURE", "FUNCTION", "TRIGGER",
		"PRIMARY", "FOREIGN", "KEY", "UNIQUE", "NOT", "NULL", "DEFAULT", "SERIAL", "BIGSERIAL",
		"AND", "OR", "IN", "LIKE", "ILIKE", "BETWEEN", "EXISTS", "IS", "CASE", "WHEN", "THEN", "ELSE",
		"GROUP", "BY", "ORDER", "HAVING", "LIMIT", "OFFSET", "UNION", "JOIN", "LEFT", "RIGHT",
		"INNER", "OUTER", "FULL", "ON", "AS", "DISTINCT", "ALL", "ASC", "DESC",
	}
}

func (d *PostgreSQLDialect) TransformSQL(sql string, targetDialect string) (string, error) {
	switch targetDialect {
	case "mysql":
		return d.transformToMySQL(sql)
	case "sqlite":
		return d.transformToSQLite(sql)
	default:
		return sql, fmt.Errorf("unsupported target dialect: %s", targetDialect)
	}
}

func (d *PostgreSQLDialect) transformToMySQL(sql string) (string, error) {
	transformed := sql

	// Replace double quotes with backticks for identifiers
	// This is a simplified transformation
	identifierPattern := regexp.MustCompile(`"([^"]+)"`)
	transformed = identifierPattern.ReplaceAllString(transformed, "`$1`")

	// Transform LIMIT OFFSET to MySQL format
	limitPattern := regexp.MustCompile(`(?i)LIMIT\s+(\d+)\s+OFFSET\s+(\d+)`)
	transformed = limitPattern.ReplaceAllString(transformed, "LIMIT $2, $1")

	return transformed, nil
}

func (d *PostgreSQLDialect) transformToSQLite(sql string) (string, error) {
	transformed := sql

	// Remove double quotes for simpler identifiers
	transformed = strings.ReplaceAll(transformed, "\"", "")

	// Replace PostgreSQL-specific functions
	transformed = strings.ReplaceAll(transformed, "CURRENT_DATE", "DATE('now')")
	transformed = strings.ReplaceAll(transformed, "NOW()", "DATETIME('now')")

	return transformed, nil
}

// SQLiteDialect implements SQLDialect for SQLite
type SQLiteDialect struct{}

func (d *SQLiteDialect) Name() string {
	return "SQLite"
}

func (d *SQLiteDialect) ValidateSQL(sql string) ([]ValidationResult, error) {
	var results []ValidationResult

	sql = strings.TrimSpace(sql)
	if sql == "" {
		return []ValidationResult{{
			Type:    "syntax",
			Level:   "error",
			Message: "Empty SQL statement",
		}}, nil
	}

	upper := strings.ToUpper(sql)

	// Check for proper statement termination
	if !strings.HasSuffix(strings.TrimSpace(sql), ";") {
		results = append(results, ValidationResult{
			Type:       "syntax",
			Level:      "warning",
			Message:    "SQL statement should end with semicolon",
			Suggestion: "Add ';' at the end of the statement",
		})
	}

	// Check for SQLite limitations
	if strings.Contains(upper, "RIGHT JOIN") || strings.Contains(upper, "FULL JOIN") {
		results = append(results, ValidationResult{
			Type:       "syntax",
			Level:      "error",
			Message:    "SQLite does not support RIGHT JOIN or FULL OUTER JOIN",
			Suggestion: "Use LEFT JOIN or restructure the query",
		})
	}

	return results, nil
}

func (d *SQLiteDialect) OptimizeSQL(sql string) (string, []string, error) {
	var suggestions []string
	optimizedSQL := sql

	upper := strings.ToUpper(sql)

	// SQLite-specific optimization suggestions
	if strings.Contains(upper, "SELECT") && !strings.Contains(upper, "LIMIT") {
		suggestions = append(suggestions, "Consider adding a LIMIT clause for better performance")
	}

	if strings.Contains(upper, "WHERE") {
		suggestions = append(suggestions, "Ensure appropriate indexes exist for WHERE clause columns")
	}

	return optimizedSQL, suggestions, nil
}

func (d *SQLiteDialect) FormatSQL(sql string) (string, error) {
	// Basic SQL formatting
	formatted := strings.TrimSpace(sql)

	keywords := []string{"SELECT", "FROM", "WHERE", "GROUP BY", "HAVING", "ORDER BY", "LIMIT"}
	for _, keyword := range keywords {
		pattern := regexp.MustCompile(`(?i)\b` + keyword + `\b`)
		formatted = pattern.ReplaceAllString(formatted, "\n"+keyword)
	}

	return strings.TrimSpace(formatted), nil
}

func (d *SQLiteDialect) GetDataTypes() []DataType {
	return []DataType{
		{Name: "INTEGER", Category: "numeric"},
		{Name: "REAL", Category: "numeric"},
		{Name: "TEXT", Category: "string"},
		{Name: "BLOB", Category: "binary"},
		{Name: "NUMERIC", Category: "numeric"},
		// SQLite is dynamically typed, but these are the storage classes
	}
}

func (d *SQLiteDialect) GetFunctions() []Function {
	return []Function{
		{Name: "COUNT", Category: "aggregate", Description: "Count rows", Syntax: "COUNT(column)", Examples: []string{"COUNT(*)", "COUNT(id)"}},
		{Name: "SUM", Category: "aggregate", Description: "Sum values", Syntax: "SUM(column)", Examples: []string{"SUM(amount)"}},
		{Name: "AVG", Category: "aggregate", Description: "Average values", Syntax: "AVG(column)", Examples: []string{"AVG(price)"}},
		{Name: "MAX", Category: "aggregate", Description: "Maximum value", Syntax: "MAX(column)", Examples: []string{"MAX(created_at)"}},
		{Name: "MIN", Category: "aggregate", Description: "Minimum value", Syntax: "MIN(column)", Examples: []string{"MIN(price)"}},
		{Name: "LENGTH", Category: "string", Description: "String length", Syntax: "LENGTH(str)", Examples: []string{"LENGTH(description)"}},
		{Name: "SUBSTR", Category: "string", Description: "Extract substring", Syntax: "SUBSTR(str, pos, len)", Examples: []string{"SUBSTR(name, 1, 10)"}},
		{Name: "DATETIME", Category: "date", Description: "Date and time function", Syntax: "DATETIME(timestring, modifier...)", Examples: []string{"DATETIME('now')", "DATETIME('2023-01-01', '+1 day')"}},
		{Name: "DATE", Category: "date", Description: "Date function", Syntax: "DATE(timestring, modifier...)", Examples: []string{"DATE('now')", "DATE('2023-01-01')"}},
		{Name: "STRFTIME", Category: "date", Description: "Format date/time", Syntax: "STRFTIME(format, timestring)", Examples: []string{"STRFTIME('%Y-%m-%d', 'now')"}},
	}
}

func (d *SQLiteDialect) GetKeywords() []string {
	return []string{
		"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER",
		"TABLE", "INDEX", "VIEW", "TRIGGER", "PRIMARY", "FOREIGN", "KEY", "UNIQUE",
		"NOT", "NULL", "DEFAULT", "AUTOINCREMENT", "AND", "OR", "IN", "LIKE", "GLOB",
		"BETWEEN", "EXISTS", "IS", "CASE", "WHEN", "THEN", "ELSE", "GROUP", "BY",
		"ORDER", "HAVING", "LIMIT", "OFFSET", "UNION", "JOIN", "LEFT", "INNER",
		"ON", "AS", "DISTINCT", "ALL", "ASC", "DESC",
	}
}

func (d *SQLiteDialect) TransformSQL(sql string, targetDialect string) (string, error) {
	switch targetDialect {
	case "mysql":
		return d.transformToMySQL(sql)
	case "postgresql":
		return d.transformToPostgreSQL(sql)
	default:
		return sql, fmt.Errorf("unsupported target dialect: %s", targetDialect)
	}
}

func (d *SQLiteDialect) transformToMySQL(sql string) (string, error) {
	transformed := sql

	// Replace SQLite date functions with MySQL equivalents
	transformed = strings.ReplaceAll(transformed, "DATETIME('now')", "NOW()")
	transformed = strings.ReplaceAll(transformed, "DATE('now')", "CURDATE()")

	// Replace SUBSTR with SUBSTRING
	substrPattern := regexp.MustCompile(`(?i)SUBSTR\s*\(\s*([^,]+),\s*([^,]+),\s*([^)]+)\s*\)`)
	transformed = substrPattern.ReplaceAllString(transformed, "SUBSTRING($1, $2, $3)")

	return transformed, nil
}

func (d *SQLiteDialect) transformToPostgreSQL(sql string) (string, error) {
	transformed := sql

	// Replace SQLite date functions with PostgreSQL equivalents
	transformed = strings.ReplaceAll(transformed, "DATETIME('now')", "NOW()")
	transformed = strings.ReplaceAll(transformed, "DATE('now')", "CURRENT_DATE")

	// Replace SUBSTR with SUBSTRING
	substrPattern := regexp.MustCompile(`(?i)SUBSTR\s*\(\s*([^,]+),\s*([^,]+),\s*([^)]+)\s*\)`)
	transformed = substrPattern.ReplaceAllString(transformed, "SUBSTRING($1 FROM $2 FOR $3)")

	return transformed, nil
}

// isKeywordUsedAsCommand checks if a keyword is used as a SQL command rather than an identifier
func isKeywordUsedAsCommand(sql, keyword string) bool {
	// This is a simplified check - in practice you'd want more sophisticated parsing
	commandPrefixes := []string{"SELECT ", "FROM ", "WHERE ", "GROUP BY", "ORDER BY", "HAVING ", "UNION ", "JOIN "}
	for _, prefix := range commandPrefixes {
		if strings.Contains(sql, prefix) && strings.Contains(sql, prefix+keyword) {
			return true
		}
	}
	return false
}