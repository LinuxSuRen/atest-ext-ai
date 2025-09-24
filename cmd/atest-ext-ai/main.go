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
)

func main() {
	// Configure memory optimization
	configureMemorySettings()

	// Setup structured logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting atest-ext-ai plugin v1.0.0")

	socketPath := getSocketPath()
	log.Printf("Using Unix socket path: %s", socketPath)

	// Clean up any existing socket file
	if err := cleanupSocketFile(socketPath); err != nil {
		log.Fatalf("Failed to cleanup existing socket file: %v", err)
	}

	// Create Unix socket listener
	listener, err := createSocketListener(socketPath)
	if err != nil {
		log.Fatalf("Failed to create socket listener at %s: %v", socketPath, err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			log.Printf("Error closing listener: %v", err)
		}
		if err := cleanupSocketFile(socketPath); err != nil {
			log.Printf("Error during socket cleanup: %v", err)
		}
		log.Println("Socket cleanup completed")
	}()

	// Initialize AI plugin service
	aiPlugin, err := plugin.NewAIPluginService()
	if err != nil {
		log.Fatalf("Failed to initialize AI plugin service: %v", err)
	}
	log.Println("AI plugin service initialized successfully")

	// Create gRPC server with enhanced configuration
	grpcServer := createGRPCServer()
	remote.RegisterLoaderServer(grpcServer, aiPlugin)
	log.Println("gRPC server configured with LoaderServer")

	// Add debug logging for all gRPC calls
	log.Println("🎯 gRPC server registered and ready to accept calls")

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-signalChan
		log.Printf("Received signal: %v, initiating graceful shutdown...", sig)

		// Shutdown AI plugin first
		aiPlugin.Shutdown()
		log.Println("AI plugin service shutdown completed")

		// Stop gRPC server gracefully with timeout
		done := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(done)
		}()

		// Force shutdown if graceful shutdown takes too long
		select {
		case <-done:
			log.Println("gRPC server shutdown completed gracefully")
		case <-time.After(30 * time.Second):
			log.Println("Forcing gRPC server shutdown due to timeout")
			grpcServer.Stop()
		}

		cancel()
	}()

	log.Printf("AI Plugin listening on Unix socket: %s", socketPath)
	log.Printf("Plugin ready to accept gRPC connections")

	// Start serving
	if err := grpcServer.Serve(listener); err != nil {
		log.Printf("gRPC server stopped: %v", err)
	}

	<-ctx.Done()
	log.Println("AI Plugin shutdown complete")
}

// getSocketPath returns the socket path from environment or default
func getSocketPath() string {
	if path := os.Getenv("AI_PLUGIN_SOCKET_PATH"); path != "" {
		log.Printf("Using socket path from environment: %s", path)
		return path
	}

	// Use /tmp path to match main server's expectation: unix:///tmp/atest-ext-ai.sock
	socketPath := "/tmp/" + SocketFileName
	log.Printf("Using default socket path: %s", socketPath)
	return socketPath
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

// createSocketListener creates and configures Unix socket listener
func createSocketListener(path string) (net.Listener, error) {
	// Ensure the parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create socket directory %s: %w", dir, err)
	}

	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("failed to create Unix socket listener: %w", err)
	}

	// Set appropriate permissions for the socket file
	// 0660 is required to allow group access for main project integration
	if err := os.Chmod(path, 0660); err != nil { //nolint:gosec // G302: Unix socket needs group access
		_ = listener.Close()
		return nil, fmt.Errorf("failed to set socket permissions: %w", err)
	}

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
