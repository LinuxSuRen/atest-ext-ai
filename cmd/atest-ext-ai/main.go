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
	"log"
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
	"github.com/linuxsuren/atest-ext-ai/pkg/plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

const (
	// SocketFileName is the socket file name for AI plugin
	SocketFileName = "atest-ext-ai.sock"
	// defaultWindowsTCPAddress is the fallback TCP address for Windows hosts
	defaultWindowsTCPAddress = "127.0.0.1:38081"
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

	// Setup structured logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("=== Starting atest-ext-ai plugin %s ===", plugin.PluginVersion)
	log.Printf("Build info: Go version %s, OS %s, Arch %s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	log.Printf("Process PID: %d", os.Getpid())

	listenCfg := resolveListenerConfig()
	log.Printf("Socket configuration: %s (%s)", listenCfg.Display(), listenCfg.network)

	// Clean up any existing socket file
	if listenCfg.isUnix {
		log.Printf("Step 1/4: Cleaning up any existing socket file...")
		if err := cleanupSocketFile(listenCfg.address); err != nil {
			log.Fatalf("FATAL: Failed to cleanup existing socket file at %s: %v\nTroubleshooting: Check file permissions and ensure no other process is using the socket", listenCfg.address, err)
		}
	} else {
		log.Printf("Step 1/4: Preparing TCP listener on %s...", listenCfg.address)
	}

	// Create listener
	log.Printf("Step 2/4: Creating %s listener...", strings.ToUpper(listenCfg.network))
	listener, err := createListener(listenCfg)
	if err != nil {
		log.Fatalf("FATAL: Failed to create listener at %s: %v\nTroubleshooting: Check address availability, permissions, and security policies", listenCfg.Display(), err)
	}
	defer func() {
		log.Println("Performing cleanup...")
		if err := listener.Close(); err != nil {
			log.Printf("Warning: Error closing listener: %v", err)
		}
		if listenCfg.isUnix {
			if err := cleanupSocketFile(listenCfg.address); err != nil {
				log.Printf("Warning: Error during socket cleanup: %v", err)
			}
		}
		log.Println("Socket cleanup completed")
	}()

	// Initialize AI plugin service
	log.Printf("Step 3/4: Initializing AI plugin service...")
	aiPlugin, err := plugin.NewAIPluginService()
	if err != nil {
		log.Panicf("FATAL: Failed to initialize AI plugin service: %v\nTroubleshooting: Check configuration file, AI service connectivity, and logs above for details", err)
	}
	log.Println("✓ AI plugin service initialized successfully")

	// Create gRPC server with enhanced configuration
	log.Printf("Step 4/4: Registering gRPC server...")
	grpcServer := createGRPCServer()
	remote.RegisterLoaderServer(grpcServer, aiPlugin)
	log.Println("✓ gRPC server configured with LoaderServer")

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-signalChan
		log.Printf("\n=== Received signal: %v, initiating graceful shutdown ===", sig)

		// Shutdown AI plugin first
		log.Println("Shutting down AI plugin service...")
		aiPlugin.Shutdown()
		log.Println("✓ AI plugin service shutdown completed")

		// Stop gRPC server gracefully with timeout
		log.Println("Stopping gRPC server...")
		done := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(done)
		}()

		// Force shutdown if graceful shutdown takes too long
		select {
		case <-done:
			log.Println("✓ gRPC server shutdown completed gracefully")
		case <-time.After(30 * time.Second):
			log.Println("⚠ Forcing gRPC server shutdown due to timeout (30s exceeded)")
			grpcServer.Stop()
		}

		cancel()
	}()

	log.Printf("\n=== Plugin startup completed successfully ===")
	log.Printf("Socket endpoint: %s", listenCfg.URI())
	log.Printf("Status: Ready to accept gRPC connections from api-testing")
	log.Printf("To test: Use api-testing to connect to %s", listenCfg.URI())
	log.Printf("\n")

	// Start serving
	if err := grpcServer.Serve(listener); err != nil {
		log.Printf("gRPC server stopped: %v", err)
	}

	<-ctx.Done()
	log.Println("\n=== AI Plugin shutdown complete ===")

}

// resolveListenerConfig determines the appropriate listener settings based on
// environment variables and operating system defaults.
func resolveListenerConfig() listenerConfig {
	// Highest priority: explicit listen address (supports tcp:// or unix://)
	if raw := os.Getenv("AI_PLUGIN_LISTEN_ADDR"); raw != "" {
		if cfg, err := parseListenAddress(raw); err == nil {
			log.Printf("Using listener configuration from AI_PLUGIN_LISTEN_ADDR: %s", cfg.URI())
			return cfg
		}
		log.Printf("Warning: invalid AI_PLUGIN_LISTEN_ADDR value '%s', falling back to OS defaults", raw)
	}

	// Windows default: TCP loopback
	if runtime.GOOS == "windows" {
		address := os.Getenv("AI_PLUGIN_TCP_ADDR")
		if address == "" {
			address = defaultWindowsTCPAddress
		}
		log.Printf("Detected Windows platform, using TCP listener at %s", address)
		return listenerConfig{
			network: "tcp",
			address: address,
			isUnix:  false,
		}
	}

	// POSIX default: Unix domain socket
	if path := os.Getenv("AI_PLUGIN_SOCKET_PATH"); path != "" {
		log.Printf("Using socket path from AI_PLUGIN_SOCKET_PATH: %s", path)
		return listenerConfig{
			network: "unix",
			address: path,
			isUnix:  true,
		}
	}

	socketPath := "/tmp/" + SocketFileName
	log.Printf("Using default Unix socket path: %s", socketPath)
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
		log.Printf("Removed existing socket file: %s", path)
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
				log.Printf("Using custom socket permissions from SOCKET_PERMISSIONS: %04o", perms)
			} else {
				log.Printf("Warning: invalid SOCKET_PERMISSIONS '%s', using default 0666: %v", permStr, err)
			}
		}

		if err := os.Chmod(cfg.address, perms); err != nil { //nolint:gosec // G302: Socket permissions configurable via env
			_ = listener.Close()
			return nil, fmt.Errorf("failed to set socket permissions to %04o: %w", perms, err)
		}

		if fileInfo, err := os.Stat(cfg.address); err == nil {
			log.Printf("Socket created successfully:")
			log.Printf("  Path: %s", cfg.address)
			log.Printf("  Permissions: %04o (%s)", fileInfo.Mode().Perm(), fileInfo.Mode().String())
			log.Printf("  Size: %d bytes", fileInfo.Size())
			log.Printf("Troubleshooting tips:")
			log.Printf("  - If connection fails with 'permission denied', check:")
			log.Printf("    1. Client process user has read/write access (permissions: %04o)", fileInfo.Mode().Perm())
			log.Printf("    2. Client process user is in the same group (or use SOCKET_PERMISSIONS=0666)")
			log.Printf("    3. SELinux/AppArmor policies allow socket access")
			log.Printf("  - Set SOCKET_PERMISSIONS environment variable to customize (e.g., SOCKET_PERMISSIONS=0666)")
		} else {
			log.Printf("Warning: could not stat socket file for diagnostics: %v", err)
		}

		return listener, nil
	}

	listener, err := net.Listen(cfg.network, cfg.address)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s listener: %w", strings.ToUpper(cfg.network), err)
	}
	log.Printf("TCP listener created successfully on %s", cfg.address)
	return listener, nil
}

// configureMemorySettings optimizes Go runtime for limited memory environments
func configureMemorySettings() {
	// Set aggressive garbage collection for memory-constrained environments
	debug.SetGCPercent(50) // More frequent GC cycles

	// Set memory limit from environment variable if available
	if memLimit := os.Getenv("GOMEMLIMIT"); memLimit != "" {
		log.Printf("Go memory limit set to: %s", memLimit)
	}

	// Limit number of OS threads to reduce memory overhead
	runtime.GOMAXPROCS(2) // Limit to 2 cores max for CI environments

	log.Printf("Memory optimization configured: GOGC=50, GOMAXPROCS=%d", runtime.GOMAXPROCS(0))
}

// createGRPCServer creates a simple gRPC server for compatibility with older clients
func createGRPCServer() *grpc.Server {
	// Debug interceptor to log all incoming gRPC calls and connection info
	unaryInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log.Printf("🔍 gRPC Call received: %s", info.FullMethod)

		// Log connection info from context
		if peer, ok := peer.FromContext(ctx); ok {
			log.Printf("🔍 Connection from: %s", peer.Addr)
		}

		resp, err := handler(ctx, req)
		if err != nil {
			log.Printf("🔍 gRPC Call %s failed: %v", info.FullMethod, err)
		} else {
			log.Printf("🔍 gRPC Call %s succeeded", info.FullMethod)
		}
		return resp, err
	}

	// Use simple gRPC server configuration for maximum compatibility
	return grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
	)
}
