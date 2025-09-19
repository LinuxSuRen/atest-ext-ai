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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher handles file system watching for configuration hot reload
type FileWatcher struct {
	watcher     *fsnotify.Watcher
	callbacks   map[string][]ConfigChangeCallback
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	debounce    time.Duration
	lastChange  time.Time
	lastChangeMu sync.RWMutex
}

// WatchConfig contains configuration for file watching
type WatchConfig struct {
	// Debounce duration to prevent multiple rapid reloads
	DebounceDelay time.Duration
	// Recursive watch subdirectories
	Recursive bool
	// File patterns to include (empty means all)
	IncludePatterns []string
	// File patterns to exclude
	ExcludePatterns []string
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher() (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	fw := &FileWatcher{
		watcher:   watcher,
		callbacks: make(map[string][]ConfigChangeCallback),
		ctx:       ctx,
		cancel:    cancel,
		debounce:  250 * time.Millisecond, // Default debounce
	}

	// Start watching in a separate goroutine
	go fw.watch()

	return fw, nil
}

// Watch starts watching the specified paths for changes
func (fw *FileWatcher) Watch(paths []string, callback ConfigChangeCallback) error {
	if callback == nil {
		return fmt.Errorf("callback cannot be nil")
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	for _, path := range paths {
		// Add the path to the watcher
		if err := fw.watcher.Add(path); err != nil {
			return WatchError{Path: path, Err: err}
		}

		// Register the callback for this path
		fw.callbacks[path] = append(fw.callbacks[path], callback)
	}

	return nil
}

// WatchWithConfig starts watching with custom configuration
func (fw *FileWatcher) WatchWithConfig(paths []string, callback ConfigChangeCallback, config WatchConfig) error {
	if callback == nil {
		return fmt.Errorf("callback cannot be nil")
	}

	// Set debounce delay
	if config.DebounceDelay > 0 {
		fw.debounce = config.DebounceDelay
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	for _, path := range paths {
		if config.Recursive {
			if err := fw.watchRecursive(path); err != nil {
				return WatchError{Path: path, Err: err}
			}
		} else {
			if err := fw.watcher.Add(path); err != nil {
				return WatchError{Path: path, Err: err}
			}
		}

		// Register the callback for this path
		fw.callbacks[path] = append(fw.callbacks[path], callback)
	}

	return nil
}

// Unwatch removes a path from watching
func (fw *FileWatcher) Unwatch(path string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if err := fw.watcher.Remove(path); err != nil {
		return WatchError{Path: path, Err: err}
	}

	// Remove callbacks for this path
	delete(fw.callbacks, path)

	return nil
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() error {
	fw.cancel()
	return fw.watcher.Close()
}

// SetDebounceDelay sets the debounce delay for change events
func (fw *FileWatcher) SetDebounceDelay(delay time.Duration) {
	fw.lastChangeMu.Lock()
	defer fw.lastChangeMu.Unlock()
	fw.debounce = delay
}

// watch is the main watching loop
func (fw *FileWatcher) watch() {
	for {
		select {
		case <-fw.ctx.Done():
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Handle file change event
			fw.handleEvent(event)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}

			// Log error but continue watching
			fmt.Printf("File watcher error: %v\n", err)
		}
	}
}

// handleEvent processes a file system event
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	// Skip if it's not a write or create event
	if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
		return
	}

	// Debounce rapid changes
	fw.lastChangeMu.Lock()
	now := time.Now()
	if now.Sub(fw.lastChange) < fw.debounce {
		fw.lastChangeMu.Unlock()
		return
	}
	fw.lastChange = now
	fw.lastChangeMu.Unlock()

	// Find callbacks for this file or its directory
	fw.mu.RLock()
	callbacks := fw.findCallbacksForEvent(event)
	fw.mu.RUnlock()

	// Execute callbacks
	for _, callback := range callbacks {
		go fw.safeExecuteCallback(callback, event)
	}
}

// findCallbacksForEvent finds all callbacks that should be executed for an event
func (fw *FileWatcher) findCallbacksForEvent(event fsnotify.Event) []ConfigChangeCallback {
	var callbacks []ConfigChangeCallback

	// Find exact path matches
	if pathCallbacks, exists := fw.callbacks[event.Name]; exists {
		callbacks = append(callbacks, pathCallbacks...)
	}

	// Find directory matches
	dir := filepath.Dir(event.Name)
	if dirCallbacks, exists := fw.callbacks[dir]; exists {
		callbacks = append(callbacks, dirCallbacks...)
	}

	// Find pattern matches
	for watchPath, pathCallbacks := range fw.callbacks {
		if matched, _ := filepath.Match(watchPath, event.Name); matched {
			callbacks = append(callbacks, pathCallbacks...)
		}
	}

	return callbacks
}

// safeExecuteCallback executes a callback with error recovery
func (fw *FileWatcher) safeExecuteCallback(callback ConfigChangeCallback, event fsnotify.Event) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in config change callback: %v\n", r)
		}
	}()

	// We don't have old/new values for file events, so pass the file path
	callback(event.Name, nil, event.Name)
}

// watchRecursive adds a path and all its subdirectories to the watcher
func (fw *FileWatcher) watchRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only watch directories
		if info.IsDir() {
			return fw.watcher.Add(path)
		}

		return nil
	})
}

// ConfigWatcher provides a high-level interface for watching configuration changes
type ConfigWatcher struct {
	fileWatcher *FileWatcher
	loader      *Loader
	validator   *Validator
	config      *Config
	configMu    sync.RWMutex
	callbacks   []ConfigChangeCallback
	callbacksMu sync.RWMutex
}

// NewConfigWatcher creates a new configuration watcher
func NewConfigWatcher(loader *Loader, validator *Validator) (*ConfigWatcher, error) {
	fileWatcher, err := NewFileWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &ConfigWatcher{
		fileWatcher: fileWatcher,
		loader:      loader,
		validator:   validator,
		config:      loader.GetConfig(),
	}, nil
}

// Start starts watching configuration files
func (cw *ConfigWatcher) Start(configPaths []string) error {
	return cw.fileWatcher.Watch(configPaths, cw.onConfigChange)
}

// StartWithConfig starts watching with custom configuration
func (cw *ConfigWatcher) StartWithConfig(configPaths []string, watchConfig WatchConfig) error {
	return cw.fileWatcher.WatchWithConfig(configPaths, cw.onConfigChange, watchConfig)
}

// Stop stops watching configuration files
func (cw *ConfigWatcher) Stop() error {
	return cw.fileWatcher.Stop()
}

// RegisterCallback registers a callback for configuration changes
func (cw *ConfigWatcher) RegisterCallback(callback ConfigChangeCallback) {
	if callback == nil {
		return
	}

	cw.callbacksMu.Lock()
	defer cw.callbacksMu.Unlock()
	cw.callbacks = append(cw.callbacks, callback)
}

// GetConfig returns the current configuration (thread-safe)
func (cw *ConfigWatcher) GetConfig() *Config {
	cw.configMu.RLock()
	defer cw.configMu.RUnlock()
	return cw.config
}

// onConfigChange handles configuration file changes
func (cw *ConfigWatcher) onConfigChange(key string, oldValue, newValue interface{}) {
	// Create a new loader to reload configuration
	newLoader := NewLoader()

	// Reload configuration from the same paths
	if err := newLoader.Load(); err != nil {
		fmt.Printf("Failed to reload configuration: %v\n", err)
		return
	}

	// Validate the new configuration
	newConfig := newLoader.GetConfig()
	if err := cw.validator.ValidateConfig(newConfig); err != nil {
		fmt.Printf("New configuration is invalid: %v\n", err)
		return
	}

	// Update current configuration
	cw.configMu.Lock()
	oldConfig := cw.config
	cw.config = newConfig
	cw.configMu.Unlock()

	// Execute registered callbacks
	cw.callbacksMu.RLock()
	callbacks := make([]ConfigChangeCallback, len(cw.callbacks))
	copy(callbacks, cw.callbacks)
	cw.callbacksMu.RUnlock()

	for _, callback := range callbacks {
		go func(cb ConfigChangeCallback) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Panic in config change callback: %v\n", r)
				}
			}()
			cb("config", oldConfig, newConfig)
		}(callback)
	}
}

// ReloadConfig manually reloads the configuration
func (cw *ConfigWatcher) ReloadConfig() error {
	// Create a new loader to reload configuration
	newLoader := NewLoader()

	// Reload configuration
	if err := newLoader.Load(); err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	// Validate the new configuration
	newConfig := newLoader.GetConfig()
	if err := cw.validator.ValidateConfig(newConfig); err != nil {
		return fmt.Errorf("new configuration is invalid: %w", err)
	}

	// Update current configuration
	cw.configMu.Lock()
	oldConfig := cw.config
	cw.config = newConfig
	cw.loader = newLoader
	cw.configMu.Unlock()

	// Execute registered callbacks
	cw.callbacksMu.RLock()
	callbacks := make([]ConfigChangeCallback, len(cw.callbacks))
	copy(callbacks, cw.callbacks)
	cw.callbacksMu.RUnlock()

	for _, callback := range callbacks {
		go func(cb ConfigChangeCallback) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Panic in config change callback: %v\n", r)
				}
			}()
			cb("config", oldConfig, newConfig)
		}(callback)
	}

	return nil
}