package constants

// Default values shared across the project to avoid hard-coded strings.
const (
	// Socket defaults
	DefaultSocketFileName       = "atest-ext-ai.sock"
	DefaultUnixSocketPath       = "/tmp/" + DefaultSocketFileName
	DefaultWindowsListenAddress = "127.0.0.1:38081"

	// Server defaults
	DefaultServerHost = "0.0.0.0"
	DefaultServerPort = 8080

	// Plugin metadata defaults
	DefaultPluginName        = "atest-ext-ai"
	DefaultPluginVersion     = "1.0.0"
	DefaultPluginEnvironment = "production"
	DefaultPluginLogLevel    = "info"

	// AI related defaults
	DefaultAIService       = "ollama"
	DefaultOllamaEndpoint  = "http://localhost:11434"
	DefaultOllamaModel     = "qwen2.5-coder:latest"
	DefaultOllamaMaxTokens = 4096
	DefaultOllamaPriority  = 1

	// Database defaults
	DefaultDatabaseDriver = "sqlite"
	DefaultDatabaseDSN    = "file:atest-ext-ai.db?cache=shared&mode=rwc"
	DefaultDatabaseType   = "mysql"

	// Logging defaults
	DefaultLoggingLevel  = "info"
	DefaultLoggingFormat = "json"
	DefaultLoggingOutput = "stdout"
	DefaultLogFilePath   = "/var/log/atest-ext-ai.log"
	DefaultLogFileSize   = "100MB"
)
