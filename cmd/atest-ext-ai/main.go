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

// Package main starts the atest-ext-ai plugin process and exposes the gRPC socket.
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/linuxsuren/api-testing/pkg/testing/remote"
	"github.com/linuxsuren/atest-ext-ai/pkg/constants"
	grpcx "github.com/linuxsuren/atest-ext-ai/pkg/grpc"
	"github.com/linuxsuren/atest-ext-ai/pkg/logging"
	"github.com/linuxsuren/atest-ext-ai/pkg/plugin"
	"google.golang.org/grpc"
)

type listenerConfig struct {
	network string
	address string
	isUnix  bool
}

func (l listenerConfig) URI() string {
	switch l.network {
	case "unix":
		path := l.address
		if !strings.HasPrefix(path, "/") {
			path = "/" + strings.TrimPrefix(path, "/")
		}
		return "unix://" + path
	default:
		return fmt.Sprintf("%s://%s", l.network, l.address)
	}
}

func (l listenerConfig) Display() string {
	if l.isUnix {
		return l.address
	}
	return l.address
}

func main() {
	// Configure memory optimization
	configureMemorySettings()

	logging.Logger.Info("Starting atest-ext-ai plugin",
		"version", plugin.PluginVersion,
		"go_version", runtime.Version(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"pid", os.Getpid())

	listenCfg := resolveListenerConfig()
	logging.Logger.Info("Socket configuration resolved",
		"address", listenCfg.Display(),
		"network", listenCfg.network)

	if listenCfg.isUnix {
		logging.Logger.Info("Step 1/4: Cleaning up Unix socket", "path", listenCfg.address)
		if err := cleanupSocketFile(listenCfg.address); err != nil {
			logging.Logger.Error("Failed to cleanup existing socket file", "path", listenCfg.address, "error", err)
			os.Exit(1)
		}
	} else {
		logging.Logger.Info("Step 1/4: Preparing TCP listener", "address", listenCfg.address)
	}

	logging.Logger.Info("Step 2/4: Creating listener", "network", strings.ToUpper(listenCfg.network))
	listener, err := createListener(listenCfg)
	if err != nil {
		logging.Logger.Error("Failed to create listener", "address", listenCfg.Display(), "error", err)
		os.Exit(1)
	}
	defer func() {
		logging.Logger.Info("Performing listener cleanup")
		if err := listener.Close(); err != nil {
			logging.Logger.Warn("Error closing listener", "error", err)
		}
		if listenCfg.isUnix {
			if err := cleanupSocketFile(listenCfg.address); err != nil {
				logging.Logger.Warn("Error during socket cleanup", "error", err)
			}
		}
		logging.Logger.Info("Socket cleanup completed")
	}()

	logging.Logger.Info("Step 3/4: Initializing AI plugin service")
	aiPlugin, err := plugin.NewAIPluginService()
	if err != nil {
		logging.Logger.Error("Failed to initialize AI plugin service", "error", err)
		panic(err)
	}
	logging.Logger.Info("AI plugin service initialized successfully")

	logging.Logger.Info("Step 4/4: Registering gRPC server")
	grpcServer := createGRPCServer()
	remote.RegisterLoaderServer(grpcServer, aiPlugin)
	logging.Logger.Info("gRPC server configured with LoaderServer")

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-signalChan
		logging.Logger.Warn("Received shutdown signal", "signal", sig.String())

		logging.Logger.Info("Shutting down AI plugin service")
		aiPlugin.Shutdown()
		logging.Logger.Info("AI plugin service shutdown completed")

		logging.Logger.Info("Stopping gRPC server")
		done := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
			logging.Logger.Info("gRPC server shutdown completed gracefully")
		case <-time.After(constants.Timeouts.Shutdown):
			logging.Logger.Warn("Forcing gRPC server shutdown due to timeout", "timeout", constants.Timeouts.Shutdown)
			grpcServer.Stop()
		}

		cancel()
	}()

	logging.Logger.Info("Plugin startup completed successfully",
		"endpoint", listenCfg.URI(),
		"status", "ready")

	if err := grpcServer.Serve(listener); err != nil {
		logging.Logger.Warn("gRPC server stopped", "error", err)
	}

	<-ctx.Done()
	logging.Logger.Info("AI Plugin shutdown complete")

}

// resolveListenerConfig determines the appropriate listener settings based on
// environment variables and operating system defaults.
func resolveListenerConfig() listenerConfig {
	// Highest priority: explicit listen address (supports tcp:// or unix://)
	if raw := os.Getenv("AI_PLUGIN_LISTEN_ADDR"); raw != "" {
		if cfg, err := parseListenAddress(raw); err == nil {
			logging.Logger.Info("Using listener configuration from AI_PLUGIN_LISTEN_ADDR", "address", cfg.URI())
			return cfg
		}
		logging.Logger.Warn("Invalid AI_PLUGIN_LISTEN_ADDR value; falling back to defaults", "value", raw)
	}

	// Windows default: TCP loopback
	if runtime.GOOS == "windows" {
		address := os.Getenv("AI_PLUGIN_TCP_ADDR")
		if address == "" {
			address = constants.DefaultWindowsListenAddress
		}
		logging.Logger.Info("Detected Windows platform, using TCP listener", "address", address)
		return listenerConfig{
			network: "tcp",
			address: address,
			isUnix:  false,
		}
	}

	// POSIX default: Unix domain socket
	if path := os.Getenv("AI_PLUGIN_SOCKET_PATH"); path != "" {
		logging.Logger.Info("Using socket path from AI_PLUGIN_SOCKET_PATH", "path", path)
		return listenerConfig{
			network: "unix",
			address: path,
			isUnix:  true,
		}
	}

	socketPath := constants.DefaultUnixSocketPath
	logging.Logger.Info("Using default Unix socket path", "path", socketPath)
	return listenerConfig{
		network: "unix",
		address: socketPath,
		isUnix:  true,
	}
}

func parseListenAddress(value string) (listenerConfig, error) {
	addr := strings.TrimSpace(value)
	if addr == "" {
		return listenerConfig{}, fmt.Errorf("empty listen address")
	}

	switch {
	case strings.HasPrefix(addr, "unix://"):
		path := strings.TrimPrefix(addr, "unix://")
		if path == "" {
			return listenerConfig{}, fmt.Errorf("unix listen address requires a path")
		}
		return listenerConfig{
			network: "unix",
			address: path,
			isUnix:  true,
		}, nil
	case strings.HasPrefix(addr, "tcp://"):
		target := strings.TrimPrefix(addr, "tcp://")
		if target == "" {
			return listenerConfig{}, fmt.Errorf("tcp listen address requires host:port")
		}
		return listenerConfig{
			network: "tcp",
			address: target,
			isUnix:  false,
		}, nil
	default:
		// Infer from simple patterns: path -> unix, host:port -> tcp
		if strings.HasPrefix(addr, "/") || strings.Contains(addr, "\\") {
			return listenerConfig{
				network: "unix",
				address: addr,
				isUnix:  true,
			}, nil
		}
		if strings.Contains(addr, ":") {
			return listenerConfig{
				network: "tcp",
				address: addr,
				isUnix:  false,
			}, nil
		}
		return listenerConfig{}, fmt.Errorf("cannot determine network type from address: %s", addr)
	}
}

// cleanupSocketFile removes existing socket file if it exists
func cleanupSocketFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove existing socket file %s: %w", path, err)
		}
		logging.Logger.Info("Removed existing socket file", "path", path)
	}
	return nil
}

// createListener creates either a Unix domain socket listener or a TCP listener
// depending on the provided configuration.
func createListener(cfg listenerConfig) (net.Listener, error) {
	if cfg.network == "unix" {
		dir := filepath.Dir(cfg.address)
		// #nosec G301 -- socket directory must remain accessible to API clients
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create socket directory %s: %w", dir, err)
		}

		listener, err := net.Listen("unix", cfg.address)
		if err != nil {
			return nil, fmt.Errorf("failed to create Unix socket listener: %w", err)
		}

		perms := os.FileMode(0666)
		if permStr := os.Getenv("SOCKET_PERMISSIONS"); permStr != "" {
			var permInt uint32
			if _, err := fmt.Sscanf(permStr, "%o", &permInt); err == nil {
				perms = os.FileMode(permInt)
				logging.Logger.Info("Using custom socket permissions from SOCKET_PERMISSIONS", "permissions", fmt.Sprintf("%04o", perms))
			} else {
				logging.Logger.Warn("Invalid SOCKET_PERMISSIONS value; using default 0666", "value", permStr, "error", err)
			}
		}

		if err := os.Chmod(cfg.address, perms); err != nil { //nolint:gosec // G302: Socket permissions configurable via env
			_ = listener.Close()
			return nil, fmt.Errorf("failed to set socket permissions to %04o: %w", perms, err)
		}

		if fileInfo, err := os.Stat(cfg.address); err == nil {
			logging.Logger.Info("Socket created successfully",
				"path", cfg.address,
				"permissions", fmt.Sprintf("%04o", fileInfo.Mode().Perm()),
				"size_bytes", fileInfo.Size())
		} else {
			logging.Logger.Warn("Could not stat socket file for diagnostics", "error", err)
		}

		return listener, nil
	}

	listener, err := net.Listen(cfg.network, cfg.address)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s listener: %w", strings.ToUpper(cfg.network), err)
	}
	logging.Logger.Info("TCP listener created", "address", cfg.address)
	return listener, nil
}

// configureMemorySettings optimizes Go runtime for limited memory environments
func configureMemorySettings() {
	// Set aggressive garbage collection for memory-constrained environments
	debug.SetGCPercent(constants.Runtime.GCPercent) // More frequent GC cycles

	// Set memory limit from environment variable if available
	if memLimit := os.Getenv("GOMEMLIMIT"); memLimit != "" {
		logging.Logger.Info("Go memory limit set", "value", memLimit)
	}

	// Limit number of OS threads to reduce memory overhead
	runtime.GOMAXPROCS(constants.Runtime.MaxProcs) // Limit OS threads for CI environments

	logging.Logger.Info("Memory optimization configured",
		"gogc", constants.Runtime.GCPercent,
		"gomaxprocs", runtime.GOMAXPROCS(0))
}

// createGRPCServer wires the standard interceptors used across the plugin.
func createGRPCServer() *grpc.Server {
	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcx.RequestIDInterceptor(),
			grpcx.LoggingInterceptor(),
		),
	)
}
