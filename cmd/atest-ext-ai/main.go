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
	log.Printf("=== Starting atest-ext-ai plugin v1.0.0 ===")
	log.Printf("Build info: Go version %s, OS %s, Arch %s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	log.Printf("Process PID: %d", os.Getpid())

	socketPath := getSocketPath()
	log.Printf("Socket configuration: %s", socketPath)

	// Clean up any existing socket file
	log.Printf("Step 1/4: Cleaning up any existing socket file...")
	if err := cleanupSocketFile(socketPath); err != nil {
		log.Fatalf("FATAL: Failed to cleanup existing socket file at %s: %v\nTroubleshooting: Check file permissions and ensure no other process is using the socket", socketPath, err)
	}

	// Create Unix socket listener
	log.Printf("Step 2/4: Creating Unix socket listener...")
	listener, err := createSocketListener(socketPath)
	if err != nil {
		log.Fatalf("FATAL: Failed to create socket listener at %s: %v\nTroubleshooting: Check directory permissions, disk space, and SELinux/AppArmor policies", socketPath, err)
	}
	defer func() {
		log.Println("Performing cleanup...")
		if err := listener.Close(); err != nil {
			log.Printf("Warning: Error closing listener: %v", err)
		}
		if err := cleanupSocketFile(socketPath); err != nil {
			log.Printf("Warning: Error during socket cleanup: %v", err)
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
	log.Printf("Socket path: %s", socketPath)
	log.Printf("Status: Ready to accept gRPC connections from api-testing")
	log.Printf("To test: Use api-testing to connect to unix://%s", socketPath)
	log.Printf("\n")

	// Start serving
	if err := grpcServer.Serve(listener); err != nil {
		log.Printf("gRPC server stopped: %v", err)
	}

	<-ctx.Done()
	log.Println("\n=== AI Plugin shutdown complete ===")

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

// createSocketListener creates and configures Unix socket listener with flexible permissions
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

	// Get socket permissions from environment or use default
	// SOCKET_PERMISSIONS can be set to customize permissions (e.g., "0666" for world-writable)
	// Default is 0666 to allow connections from different users/groups
	perms := os.FileMode(0666)
	if permStr := os.Getenv("SOCKET_PERMISSIONS"); permStr != "" {
		// Parse octal permission string (e.g., "0660", "0666")
		var permInt uint32
		if _, err := fmt.Sscanf(permStr, "%o", &permInt); err == nil {
			perms = os.FileMode(permInt)
			log.Printf("Using custom socket permissions from SOCKET_PERMISSIONS: %04o", perms)
		} else {
			log.Printf("Warning: invalid SOCKET_PERMISSIONS '%s', using default 0666: %v", permStr, err)
		}
	}

	// Set socket permissions
	if err := os.Chmod(path, perms); err != nil { //nolint:gosec // G302: Socket permissions configurable via env
		_ = listener.Close()
		return nil, fmt.Errorf("failed to set socket permissions to %04o: %w", perms, err)
	}

	// Print diagnostic information for troubleshooting
	if fileInfo, err := os.Stat(path); err == nil {
		log.Printf("Socket created successfully:")
		log.Printf("  Path: %s", path)
		log.Printf("  Permissions: %04o (%s)", fileInfo.Mode().Perm(), fileInfo.Mode().String())
		log.Printf("  Size: %d bytes", fileInfo.Size())

		// Print owner information if possible (Unix-specific)
		if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
			log.Printf("  Owner UID: %d, GID: %d", stat.Uid, stat.Gid)
		}

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
