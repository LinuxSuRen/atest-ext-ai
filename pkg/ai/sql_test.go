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
	"strings"
	"testing"
)

func TestMySQLDialect_Name(t *testing.T) {
	dialect := &MySQLDialect{}
	expected := "MySQL"
	if dialect.Name() != expected {
		t.Errorf("Expected %s, got %s", expected, dialect.Name())
	}
}

func TestMySQLDialect_ValidateSQL(t *testing.T) {
	dialect := &MySQLDialect{}

	tests := []struct {
		name          string
		sql           string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "valid SQL with semicolon",
			sql:           "SELECT * FROM users;",
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "valid SQL without semicolon",
			sql:           "SELECT * FROM users",
			expectedCount: 1, // Warning about missing semicolon
			expectError:   false,
		},
		{
			name:          "empty SQL",
			sql:           "",
			expectedCount: 1, // Error for empty statement
			expectError:   false,
		},
		{
			name:          "SQL with reserved keyword",
			sql:           "SELECT * FROM `order`;",
			expectedCount: 0, // Using backticks, so should be OK
			expectError:   false,
		},
		{
			name:          "SQL with valid LIMIT",
			sql:           "SELECT * FROM users LIMIT 10;",
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "SQL with MySQL-style LIMIT",
			sql:           "SELECT * FROM users LIMIT 10, 20;",
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := dialect.ValidateSQL(tt.sql)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d validation results, got %d", tt.expectedCount, len(results))
				for i, result := range results {
					t.Logf("  Result %d: %s [%s] %s", i+1, result.Type, result.Level, result.Message)
				}
			}
		})
	}
}

func TestMySQLDialect_OptimizeSQL(t *testing.T) {
	dialect := &MySQLDialect{}

	tests := []struct {
		name            string
		sql             string
		expectedSQL     string
		minSuggestions  int
	}{
		{
			name:           "SELECT without LIMIT or WHERE",
			sql:            "SELECT * FROM users",
			expectedSQL:    "SELECT * FROM users", // No change expected
			minSuggestions: 1,                     // Should suggest LIMIT
		},
		{
			name:           "SELECT with WHERE clause",
			sql:            "SELECT * FROM users WHERE age > 18",
			expectedSQL:    "SELECT * FROM users WHERE age > 18",
			minSuggestions: 1, // Should suggest indexes
		},
		{
			name:           "SELECT with subquery using IN",
			sql:            "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)",
			expectedSQL:    "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)",
			minSuggestions: 2, // Should suggest EXISTS and indexes
		},
		{
			name:           "SELECT with LIMIT",
			sql:            "SELECT * FROM users LIMIT 10",
			expectedSQL:    "SELECT * FROM users LIMIT 10",
			minSuggestions: 0, // Might have suggestions but not required
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimizedSQL, suggestions, err := dialect.OptimizeSQL(tt.sql)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if optimizedSQL != tt.expectedSQL {
				t.Errorf("Expected optimized SQL: %s, got: %s", tt.expectedSQL, optimizedSQL)
			}

			if len(suggestions) < tt.minSuggestions {
				t.Errorf("Expected at least %d suggestions, got %d", tt.minSuggestions, len(suggestions))
				for i, suggestion := range suggestions {
					t.Logf("  Suggestion %d: %s", i+1, suggestion)
				}
			}
		})
	}
}

func TestMySQLDialect_GetDataTypes(t *testing.T) {
	dialect := &MySQLDialect{}
	dataTypes := dialect.GetDataTypes()

	if len(dataTypes) == 0 {
		t.Errorf("Expected data types but got none")
	}

	// Check for key data types
	expectedTypes := []string{"INT", "VARCHAR", "TEXT", "DATETIME", "DECIMAL"}
	for _, expected := range expectedTypes {
		found := false
		for _, dataType := range dataTypes {
			if dataType.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected data type %s not found", expected)
		}
	}
}

func TestMySQLDialect_GetFunctions(t *testing.T) {
	dialect := &MySQLDialect{}
	functions := dialect.GetFunctions()

	if len(functions) == 0 {
		t.Errorf("Expected functions but got none")
	}

	// Check for key functions
	expectedFunctions := []string{"COUNT", "SUM", "AVG", "CONCAT", "NOW"}
	for _, expected := range expectedFunctions {
		found := false
		for _, function := range functions {
			if function.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected function %s not found", expected)
		}
	}
}

func TestMySQLDialect_TransformSQL(t *testing.T) {
	dialect := &MySQLDialect{}

	tests := []struct {
		name         string
		sql          string
		targetDialect string
		expectedSQL  string
		expectError  bool
	}{
		{
			name:          "MySQL to PostgreSQL - backticks",
			sql:           "SELECT `name` FROM `users`",
			targetDialect: "postgresql",
			expectedSQL:   "SELECT \"NAME\" FROM \"USERS\"",
			expectError:   false,
		},
		{
			name:          "MySQL to PostgreSQL - LIMIT offset",
			sql:           "SELECT * FROM users LIMIT 10, 20",
			targetDialect: "postgresql",
			expectedSQL:   "SELECT * FROM USERS LIMIT 20 OFFSET 10",
			expectError:   false,
		},
		{
			name:          "MySQL to SQLite - remove backticks",
			sql:           "SELECT `name` FROM `users`",
			targetDialect: "sqlite",
			expectedSQL:   "SELECT NAME FROM USERS",
			expectError:   false,
		},
		{
			name:          "MySQL to SQLite - NOW() function",
			sql:           "SELECT NOW() FROM users",
			targetDialect: "sqlite",
			expectedSQL:   "SELECT DATETIME('now') FROM USERS",
			expectError:   false,
		},
		{
			name:          "unsupported target dialect",
			sql:           "SELECT * FROM users",
			targetDialect: "oracle",
			expectedSQL:   "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dialect.TransformSQL(tt.sql, tt.targetDialect)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expectedSQL {
				t.Errorf("Expected transformed SQL: %s, got: %s", tt.expectedSQL, result)
			}
		})
	}
}

func TestPostgreSQLDialect_Name(t *testing.T) {
	dialect := &PostgreSQLDialect{}
	expected := "PostgreSQL"
	if dialect.Name() != expected {
		t.Errorf("Expected %s, got %s", expected, dialect.Name())
	}
}

func TestPostgreSQLDialect_ValidateSQL(t *testing.T) {
	dialect := &PostgreSQLDialect{}

	tests := []struct {
		name          string
		sql           string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "valid PostgreSQL SQL",
			sql:           "SELECT * FROM users;",
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "MySQL-style LIMIT",
			sql:           "SELECT * FROM users LIMIT 10, 20;",
			expectedCount: 1, // Error for MySQL-style LIMIT
			expectError:   false,
		},
		{
			name:          "backticks instead of double quotes",
			sql:           "SELECT `name` FROM `users`;",
			expectedCount: 1, // Warning about backticks
			expectError:   false,
		},
		{
			name:          "valid PostgreSQL LIMIT",
			sql:           "SELECT * FROM users LIMIT 20 OFFSET 10;",
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := dialect.ValidateSQL(tt.sql)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d validation results, got %d", tt.expectedCount, len(results))
				for i, result := range results {
					t.Logf("  Result %d: %s [%s] %s", i+1, result.Type, result.Level, result.Message)
				}
			}
		})
	}
}

func TestPostgreSQLDialect_GetDataTypes(t *testing.T) {
	dialect := &PostgreSQLDialect{}
	dataTypes := dialect.GetDataTypes()

	if len(dataTypes) == 0 {
		t.Errorf("Expected data types but got none")
	}

	// Check for PostgreSQL-specific data types
	expectedTypes := []string{"INTEGER", "BIGINT", "VARCHAR", "TEXT", "TIMESTAMP", "JSONB", "UUID", "SERIAL"}
	for _, expected := range expectedTypes {
		found := false
		for _, dataType := range dataTypes {
			if dataType.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected data type %s not found", expected)
		}
	}
}

func TestPostgreSQLDialect_TransformSQL(t *testing.T) {
	dialect := &PostgreSQLDialect{}

	tests := []struct {
		name          string
		sql           string
		targetDialect string
		expectedSQL   string
		expectError   bool
	}{
		{
			name:          "PostgreSQL to MySQL - double quotes to backticks",
			sql:           "SELECT \"name\" FROM \"users\"",
			targetDialect: "mysql",
			expectedSQL:   "SELECT `name` FROM `users`",
			expectError:   false,
		},
		{
			name:          "PostgreSQL to MySQL - LIMIT OFFSET",
			sql:           "SELECT * FROM users LIMIT 20 OFFSET 10",
			targetDialect: "mysql",
			expectedSQL:   "SELECT * FROM users LIMIT 10, 20",
			expectError:   false,
		},
		{
			name:          "PostgreSQL to SQLite - remove quotes",
			sql:           "SELECT \"name\" FROM \"users\"",
			targetDialect: "sqlite",
			expectedSQL:   "SELECT name FROM users",
			expectError:   false,
		},
		{
			name:          "PostgreSQL to SQLite - date functions",
			sql:           "SELECT CURRENT_DATE, NOW() FROM users",
			targetDialect: "sqlite",
			expectedSQL:   "SELECT DATE('now'), DATETIME('now') FROM users",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dialect.TransformSQL(tt.sql, tt.targetDialect)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expectedSQL {
				t.Errorf("Expected transformed SQL: %s, got: %s", tt.expectedSQL, result)
			}
		})
	}
}

func TestSQLiteDialect_Name(t *testing.T) {
	dialect := &SQLiteDialect{}
	expected := "SQLite"
	if dialect.Name() != expected {
		t.Errorf("Expected %s, got %s", expected, dialect.Name())
	}
}

func TestSQLiteDialect_ValidateSQL(t *testing.T) {
	dialect := &SQLiteDialect{}

	tests := []struct {
		name          string
		sql           string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "valid SQLite SQL",
			sql:           "SELECT * FROM users;",
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "RIGHT JOIN not supported",
			sql:           "SELECT * FROM users RIGHT JOIN orders ON users.id = orders.user_id;",
			expectedCount: 1, // Error for unsupported RIGHT JOIN
			expectError:   false,
		},
		{
			name:          "FULL OUTER JOIN not supported",
			sql:           "SELECT * FROM users FULL JOIN orders ON users.id = orders.user_id;",
			expectedCount: 1, // Error for unsupported FULL JOIN
			expectError:   false,
		},
		{
			name:          "LEFT JOIN is supported",
			sql:           "SELECT * FROM users LEFT JOIN orders ON users.id = orders.user_id;",
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := dialect.ValidateSQL(tt.sql)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d validation results, got %d", tt.expectedCount, len(results))
				for i, result := range results {
					t.Logf("  Result %d: %s [%s] %s", i+1, result.Type, result.Level, result.Message)
				}
			}
		})
	}
}

func TestSQLiteDialect_GetDataTypes(t *testing.T) {
	dialect := &SQLiteDialect{}
	dataTypes := dialect.GetDataTypes()

	if len(dataTypes) == 0 {
		t.Errorf("Expected data types but got none")
	}

	// SQLite has a limited set of storage classes
	expectedTypes := []string{"INTEGER", "REAL", "TEXT", "BLOB", "NUMERIC"}
	for _, expected := range expectedTypes {
		found := false
		for _, dataType := range dataTypes {
			if dataType.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected data type %s not found", expected)
		}
	}
}

func TestSQLiteDialect_GetFunctions(t *testing.T) {
	dialect := &SQLiteDialect{}
	functions := dialect.GetFunctions()

	if len(functions) == 0 {
		t.Errorf("Expected functions but got none")
	}

	// Check for SQLite-specific functions
	expectedFunctions := []string{"COUNT", "SUM", "LENGTH", "SUBSTR", "DATETIME", "DATE", "STRFTIME"}
	for _, expected := range expectedFunctions {
		found := false
		for _, function := range functions {
			if function.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected function %s not found", expected)
		}
	}
}

func TestSQLiteDialect_TransformSQL(t *testing.T) {
	dialect := &SQLiteDialect{}

	tests := []struct {
		name          string
		sql           string
		targetDialect string
		expectedSQL   string
		expectError   bool
	}{
		{
			name:          "SQLite to MySQL - date functions",
			sql:           "SELECT DATETIME('now'), DATE('now') FROM users",
			targetDialect: "mysql",
			expectedSQL:   "SELECT NOW(), CURDATE() FROM users",
			expectError:   false,
		},
		{
			name:          "SQLite to MySQL - SUBSTR to SUBSTRING",
			sql:           "SELECT SUBSTR(name, 1, 10) FROM users",
			targetDialect: "mysql",
			expectedSQL:   "SELECT SUBSTRING(name, 1, 10) FROM users",
			expectError:   false,
		},
		{
			name:          "SQLite to PostgreSQL - date functions",
			sql:           "SELECT DATETIME('now'), DATE('now') FROM users",
			targetDialect: "postgresql",
			expectedSQL:   "SELECT NOW(), CURRENT_DATE FROM users",
			expectError:   false,
		},
		{
			name:          "SQLite to PostgreSQL - SUBSTR syntax",
			sql:           "SELECT SUBSTR(name, 1, 10) FROM users",
			targetDialect: "postgresql",
			expectedSQL:   "SELECT SUBSTRING(name FROM 1 FOR 10) FROM users",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dialect.TransformSQL(tt.sql, tt.targetDialect)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expectedSQL {
				t.Errorf("Expected transformed SQL: %s, got: %s", tt.expectedSQL, result)
			}
		})
	}
}

func TestSQLDialect_FormatSQL(t *testing.T) {
	dialects := []struct {
		name    string
		dialect SQLDialect
	}{
		{"MySQL", &MySQLDialect{}},
		{"PostgreSQL", &PostgreSQLDialect{}},
		{"SQLite", &SQLiteDialect{}},
	}

	sql := "SELECT name, email FROM users WHERE age > 18 GROUP BY name ORDER BY name LIMIT 10"

	for _, d := range dialects {
		t.Run(d.name, func(t *testing.T) {
			formatted, err := d.dialect.FormatSQL(sql)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if formatted == "" {
				t.Errorf("Expected formatted SQL but got empty string")
				return
			}

			// Check that the formatted SQL contains line breaks for major keywords
			if !containsKeywordOnNewLine(formatted, "SELECT") {
				t.Errorf("Expected SELECT on new line in formatted SQL: %s", formatted)
			}

			if !containsKeywordOnNewLine(formatted, "FROM") {
				t.Errorf("Expected FROM on new line in formatted SQL: %s", formatted)
			}

			if !containsKeywordOnNewLine(formatted, "WHERE") {
				t.Errorf("Expected WHERE on new line in formatted SQL: %s", formatted)
			}
		})
	}
}

func containsKeywordOnNewLine(sql, keyword string) bool {
	return strings.Contains(sql, "\n"+keyword)
}

func TestSQLDialect_Integration(t *testing.T) {
	// Integration test to verify all dialects work together
	dialects := map[string]SQLDialect{
		"mysql":      &MySQLDialect{},
		"postgresql": &PostgreSQLDialect{},
		"sqlite":     &SQLiteDialect{},
	}

	originalSQL := "SELECT name FROM users WHERE age > 18"

	// Test cross-dialect transformation
	for sourceName, sourceDialect := range dialects {
		for targetName, _ := range dialects {
			if sourceName == targetName {
				continue
			}

			t.Run(sourceName+"_to_"+targetName, func(t *testing.T) {
				transformed, err := sourceDialect.TransformSQL(originalSQL, targetName)

				if err != nil {
					t.Errorf("Failed to transform from %s to %s: %v", sourceName, targetName, err)
					return
				}

				if transformed == "" {
					t.Errorf("Transform from %s to %s resulted in empty SQL", sourceName, targetName)
				}

				t.Logf("Transform %s -> %s: %s -> %s", sourceName, targetName, originalSQL, transformed)
			})
		}
	}
}