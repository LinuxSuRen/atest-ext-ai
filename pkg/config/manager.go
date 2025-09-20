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

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// Manager is the main configuration manager that orchestrates all components
type Manager struct {
	loader      *Loader
	validator   *Validator
	watcher     *ConfigWatcher
	config      *Config
	mu          sync.RWMutex
	callbacks   []ConfigChangeCallback
	callbacksMu sync.RWMutex
	isWatching  bool
	watchPaths  []string
	options     ManagerOptions
}

// ManagerOptions contains configuration manager options
type ManagerOptions struct {
	// EnableHotReload enables automatic configuration reloading
	EnableHotReload bool
	// WatchConfig contains file watching configuration
	WatchConfig WatchConfig
	// ValidateOnLoad validates configuration immediately after loading
	ValidateOnLoad bool
	// BackupConfig creates backup copies of configuration
	BackupConfig bool
	// BackupDir directory for backup files
	BackupDir string
	// AutoSave automatically saves configuration changes
	AutoSave bool
}

// DefaultManagerOptions returns default manager options
func DefaultManagerOptions() ManagerOptions {
	return ManagerOptions{
		EnableHotReload: true,
		WatchConfig: WatchConfig{
			DebounceDelay:   500 * time.Millisecond,
			Recursive:       false,
			IncludePatterns: []string{"*.yaml", "*.yml", "*.json", "*.toml"},
			ExcludePatterns: []string{"*.tmp", "*.bak", "*~"},
		},
		ValidateOnLoad: true,
		BackupConfig:   false,
		AutoSave:       false,
	}
}

// NewManager creates a new configuration manager
func NewManager(options ...ManagerOptions) (*Manager, error) {
	opts := DefaultManagerOptions()
	if len(options) > 0 {
		opts = options[0]
	}

	loader := NewLoader()
	validator := NewValidator()

	var watcher *ConfigWatcher
	if opts.EnableHotReload {
		var err error
		watcher, err = NewConfigWatcher(loader, validator)
		if err != nil {
			return nil, fmt.Errorf("failed to create config watcher: %w", err)
		}
	}

	return &Manager{
		loader:    loader,
		validator: validator,
		watcher:   watcher,
		options:   opts,
	}, nil
}

// Load loads configuration from the specified paths
func (m *Manager) Load(paths ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store watch paths for hot reload
	if len(paths) > 0 {
		m.watchPaths = paths
	}

	// Load configuration
	if err := m.loader.Load(paths...); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get loaded configuration
	m.config = m.loader.GetConfig()

	// Validate if enabled
	if m.options.ValidateOnLoad {
		if err := m.validator.ValidateConfig(m.config); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	// Create backup if enabled
	if m.options.BackupConfig {
		if err := m.createBackup(); err != nil {
			return fmt.Errorf("failed to create configuration backup: %w", err)
		}
	}

	// Start watching if hot reload is enabled
	if m.options.EnableHotReload && m.watcher != nil && !m.isWatching {
		if err := m.startWatching(); err != nil {
			return fmt.Errorf("failed to start configuration watching: %w", err)
		}
	}

	return nil
}

// LoadFromFile loads configuration from a specific file
func (m *Manager) LoadFromFile(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store watch paths
	m.watchPaths = []string{filepath.Dir(filePath)}

	// Load configuration
	if err := m.loader.LoadFromFile(filePath); err != nil {
		return fmt.Errorf("failed to load configuration from file: %w", err)
	}

	// Get loaded configuration
	m.config = m.loader.GetConfig()

	// Validate if enabled
	if m.options.ValidateOnLoad {
		if err := m.validator.ValidateConfig(m.config); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	return nil
}

// LoadFromBytes loads configuration from byte data
func (m *Manager) LoadFromBytes(data []byte, format string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load configuration
	if err := m.loader.LoadFromBytes(data, format); err != nil {
		return fmt.Errorf("failed to load configuration from bytes: %w", err)
	}

	// Get loaded configuration
	m.config = m.loader.GetConfig()

	// Validate if enabled
	if m.options.ValidateOnLoad {
		if err := m.validator.ValidateConfig(m.config); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	return nil
}

// Get retrieves a configuration value by key
func (m *Manager) Get(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.loader == nil {
		return nil
	}

	return m.loader.GetViper().Get(key)
}

// Set sets a configuration value by key
func (m *Manager) Set(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.loader == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Store old value for callback
	oldValue := m.loader.GetViper().Get(key)

	// Set the value
	m.loader.GetViper().Set(key, value)

	// Unmarshal updated configuration
	if err := m.loader.GetViper().Unmarshal(m.config); err != nil {
		return fmt.Errorf("failed to unmarshal updated configuration: %w", err)
	}

	// Validate the updated configuration
	if err := m.validator.ValidateConfig(m.config); err != nil {
		// Rollback if validation fails
		m.loader.GetViper().Set(key, oldValue)
		if err := m.loader.GetViper().Unmarshal(m.config); err != nil {
			// Log rollback error but don't return it as it would mask the validation error
		}
		return fmt.Errorf("configuration validation failed after update: %w", err)
	}

	// Auto-save if enabled
	if m.options.AutoSave {
		if err := m.save(); err != nil {
			return fmt.Errorf("failed to auto-save configuration: %w", err)
		}
	}

	// Execute callbacks
	m.executeCallbacks(key, oldValue, value)

	return nil
}

// Reload reloads the configuration from the source
func (m *Manager) Reload() error {
	return m.Load(m.watchPaths...)
}

// Watch registers a callback for configuration changes
func (m *Manager) Watch(callback ConfigChangeCallback) error {
	if callback == nil {
		return fmt.Errorf("callback cannot be nil")
	}

	m.callbacksMu.Lock()
	defer m.callbacksMu.Unlock()

	m.callbacks = append(m.callbacks, callback)

	// Also register with the watcher if available
	if m.watcher != nil {
		m.watcher.RegisterCallback(callback)
	}

	return nil
}

// Validate validates the current configuration
func (m *Manager) Validate() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	return m.validator.ValidateConfig(m.config)
}

// Export exports configuration in the specified format
func (m *Manager) Export(format string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.loader == nil {
		return nil, fmt.Errorf("no configuration loaded")
	}

	return m.loader.Export(format)
}

// GetConfig returns the complete configuration
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Stop stops the configuration manager and cleans up resources
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.watcher != nil && m.isWatching {
		if err := m.watcher.Stop(); err != nil {
			return fmt.Errorf("failed to stop configuration watcher: %w", err)
		}
		m.isWatching = false
	}

	return nil
}

// Save saves the current configuration to file
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.save()
}

// SaveToFile saves the configuration to a specific file
func (m *Manager) SaveToFile(filePath, format string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return fmt.Errorf("no configuration to save")
	}

	data, err := m.loader.Export(format)
	if err != nil {
		return fmt.Errorf("failed to export configuration: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

// GetDefaults returns a configuration with default values
func (m *Manager) GetDefaults() *Config {
	// Create a new viper instance with defaults
	v := viper.New()
	setDefaults(v)

	// Create config and unmarshal defaults
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		// Return empty config if unmarshal fails
		return &Config{}
	}

	return config
}

// Merge merges another configuration into the current one
func (m *Manager) Merge(other *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if other == nil {
		return fmt.Errorf("cannot merge nil configuration")
	}

	// Store old config for callback
	oldConfig := *m.config

	// Merge configurations
	if err := m.loader.Merge(other); err != nil {
		return fmt.Errorf("failed to merge configuration: %w", err)
	}

	// Update current config
	m.config = m.loader.GetConfig()

	// Validate merged configuration
	if err := m.validator.ValidateConfig(m.config); err != nil {
		return fmt.Errorf("merged configuration validation failed: %w", err)
	}

	// Execute callbacks
	m.executeCallbacks("config", &oldConfig, m.config)

	return nil
}

// GetSection returns a specific configuration section
func (m *Manager) GetSection(section string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return nil, fmt.Errorf("no configuration loaded")
	}

	switch section {
	case "server":
		return m.config.Server, nil
	case "plugin":
		return m.config.Plugin, nil
	case "ai":
		return m.config.AI, nil
	case "database":
		return m.config.Database, nil
	case "logging":
		return m.config.Logging, nil
	default:
		return m.Get(section), nil
	}
}

// IsWatching returns whether the configuration is being watched for changes
func (m *Manager) IsWatching() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isWatching
}

// GetWatchPaths returns the paths being watched
func (m *Manager) GetWatchPaths() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string(nil), m.watchPaths...) // Return a copy
}

// startWatching starts configuration file watching
func (m *Manager) startWatching() error {
	if len(m.watchPaths) == 0 {
		return fmt.Errorf("no paths to watch")
	}

	// Register self as callback for configuration changes
	m.watcher.RegisterCallback(m.onConfigurationChange)

	// Start watching
	if err := m.watcher.StartWithConfig(m.watchPaths, m.options.WatchConfig); err != nil {
		return err
	}

	m.isWatching = true
	return nil
}

// onConfigurationChange handles configuration changes from the watcher
func (m *Manager) onConfigurationChange(key string, oldValue, newValue interface{}) {
	// The watcher already handles reloading and validation
	// Just update our reference to the configuration
	m.mu.Lock()
	m.config = m.watcher.GetConfig()
	m.mu.Unlock()

	// Execute our registered callbacks
	m.executeCallbacks(key, oldValue, newValue)
}

// executeCallbacks executes all registered callbacks
func (m *Manager) executeCallbacks(key string, oldValue, newValue interface{}) {
	m.callbacksMu.RLock()
	callbacks := make([]ConfigChangeCallback, len(m.callbacks))
	copy(callbacks, m.callbacks)
	m.callbacksMu.RUnlock()

	for _, callback := range callbacks {
		go func(cb ConfigChangeCallback) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Panic in config change callback: %v\n", r)
				}
			}()
			cb(key, oldValue, newValue)
		}(callback)
	}
}

// save saves the current configuration (internal method without locking)
func (m *Manager) save() error {
	if m.config == nil {
		return fmt.Errorf("no configuration to save")
	}

	// Export configuration
	data, err := m.loader.Export("yaml")
	if err != nil {
		return fmt.Errorf("failed to export configuration: %w", err)
	}

	// Find the first writable path
	configPath := ""
	for _, path := range m.watchPaths {
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() {
				configPath = filepath.Join(path, "config.yaml")
			} else {
				configPath = path
			}
			break
		}
	}

	if configPath == "" {
		configPath = "./config.yaml" // Default fallback
	}

	// Write configuration file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

// createBackup creates a backup of the current configuration
func (m *Manager) createBackup() error {
	if m.config == nil {
		return nil // No config to backup
	}

	backupDir := m.options.BackupDir
	if backupDir == "" {
		backupDir = "./config/backup"
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("config_backup_%s.yaml", timestamp))

	// Export configuration
	data, err := m.loader.Export("yaml")
	if err != nil {
		return fmt.Errorf("failed to export configuration for backup: %w", err)
	}

	// Write backup file
	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// RestoreFromBackup restores configuration from a backup file
func (m *Manager) RestoreFromBackup(backupFile string) error {
	return m.LoadFromFile(backupFile)
}

// ListBackups lists available configuration backup files
func (m *Manager) ListBackups() ([]string, error) {
	backupDir := m.options.BackupDir
	if backupDir == "" {
		backupDir = "./config/backup"
	}

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			backups = append(backups, filepath.Join(backupDir, entry.Name()))
		}
	}

	return backups, nil
}

// GetStats returns configuration manager statistics
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"config_loaded":  m.config != nil,
		"is_watching":    m.isWatching,
		"watch_paths":    m.watchPaths,
		"callback_count": len(m.callbacks),
		"options":        m.options,
	}

	if m.config != nil {
		stats["ai_services_count"] = len(m.config.AI.Services)
		stats["default_ai_service"] = m.config.AI.DefaultService
	}

	return stats
}

// ValidateField validates a specific configuration field
func (m *Manager) ValidateField(fieldPath string, value interface{}) error {
	// Create a temporary struct to validate the field
	tempStruct := struct {
		Value interface{} `validate:"required"`
	}{
		Value: value,
	}

	return m.validator.ValidateStruct(tempStruct)
}

// GetConfigAsJSON returns the configuration as JSON string
func (m *Manager) GetConfigAsJSON() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return "", fmt.Errorf("no configuration loaded")
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal configuration to JSON: %w", err)
	}

	return string(data), nil
}
