package config

import (
	"os"
	"testing"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clear environment variables
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
		return
	}

	// Check default values
	if cfg.Server.Host != "localhost" {
		t.Errorf("Server.Host = %v, want %v", cfg.Server.Host, "localhost")
	}
	if cfg.Server.Port != 50051 {
		t.Errorf("Server.Port = %v, want %v", cfg.Server.Port, 50051)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %v, want %v", cfg.Database.Host, "localhost")
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %v, want %v", cfg.Database.Port, 5432)
	}
	if cfg.Database.DBName != "ai_plugin" {
		t.Errorf("Database.DBName = %v, want %v", cfg.Database.DBName, "ai_plugin")
	}
	if cfg.AI.DefaultModel != "gpt-3.5-turbo" {
		t.Errorf("AI.DefaultModel = %v, want %v", cfg.AI.DefaultModel, "gpt-3.5-turbo")
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level = %v, want %v", cfg.Logging.Level, "info")
	}
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("SERVER_HOST", "0.0.0.0")
	os.Setenv("SERVER_PORT", "8080")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "3306")
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("AI_MODEL", "gpt-4")
	os.Setenv("LOG_LEVEL", "debug")

	defer func() {
		os.Clearenv()
	}()

	cfg, err := Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
		return
	}

	// Check environment variable values
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %v, want %v", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %v, want %v", cfg.Server.Port, 8080)
	}
	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %v, want %v", cfg.Database.Host, "db.example.com")
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("Database.Port = %v, want %v", cfg.Database.Port, 3306)
	}
	if cfg.Database.DBName != "test_db" {
		t.Errorf("Database.DBName = %v, want %v", cfg.Database.DBName, "test_db")
	}
	if cfg.AI.DefaultModel != "gpt-4" {
		t.Errorf("AI.DefaultModel = %v, want %v", cfg.AI.DefaultModel, "gpt-4")
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %v, want %v", cfg.Logging.Level, "debug")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 50051,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			DBName:   "ai_plugin",
			User:     "postgres",
			Password: "password",
			SSLMode:  "disable",
		},
		AI: AIConfig{
			DefaultModel: "gpt-3.5-turbo",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestValidate_InvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "invalid server port",
			config: &Config{
				Server:   ServerConfig{Port: -1},
				Database: DatabaseConfig{Port: 5432, DBName: "test", User: "user"},
				AI:       AIConfig{DefaultModel: "gpt-3.5-turbo"},
				Logging:  LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
		},
		{
			name: "invalid database port",
			config: &Config{
				Server:   ServerConfig{Port: 50051},
				Database: DatabaseConfig{Port: 70000, DBName: "test", User: "user"},
				AI:       AIConfig{DefaultModel: "gpt-3.5-turbo"},
				Logging:  LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
		},
		{
			name: "missing database name",
			config: &Config{
				Server:   ServerConfig{Port: 50051},
				Database: DatabaseConfig{Port: 5432, DBName: "", User: "user"},
				AI:       AIConfig{DefaultModel: "gpt-3.5-turbo"},
				Logging:  LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
		},
		{
			name: "missing AI default model",
			config: &Config{
				Server:   ServerConfig{Port: 50051},
				Database: DatabaseConfig{Port: 5432, DBName: "test", User: "user"},
				AI:       AIConfig{DefaultModel: ""},
				Logging:  LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: &Config{
				Server:   ServerConfig{Port: 50051},
				Database: DatabaseConfig{Port: 5432, DBName: "test", User: "user"},
				AI:       AIConfig{DefaultModel: "gpt-3.5-turbo"},
				Logging:  LoggingConfig{Level: "invalid", Format: "text"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		default_ string
		envValue string
		want     string
	}{
		{
			name:     "environment variable exists",
			key:      "TEST_ENV_VAR",
			default_: "default",
			envValue: "env_value",
			want:     "env_value",
		},
		{
			name:     "environment variable does not exist",
			key:      "NON_EXISTENT_VAR",
			default_: "default",
			envValue: "",
			want:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnv(tt.key, tt.default_)
			if got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		default_ int
		envValue string
		want     int
	}{
		{
			name:     "valid integer",
			key:      "TEST_INT_VAR",
			default_: 100,
			envValue: "200",
			want:     200,
		},
		{
			name:     "invalid integer",
			key:      "TEST_INT_VAR",
			default_: 100,
			envValue: "invalid",
			want:     100,
		},
		{
			name:     "empty value",
			key:      "TEST_INT_VAR",
			default_: 100,
			envValue: "",
			want:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnvAsInt(tt.key, tt.default_)
			if got != tt.want {
				t.Errorf("getEnvAsInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
