/*
Copyright 2023-2025 API Testing Authors.

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
	"syscall"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/plugin"
	"github.com/linuxsuren/api-testing/pkg/testing/remote"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	// DefaultSocketPath is the standard Unix socket path for AI plugin
	DefaultSocketPath = "/tmp/atest-store-ai.sock"
)

func main() {
	// Setup structured logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting atest-store-ai plugin v1.0.0")

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
		cleanupSocketFile(socketPath)
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
		return path
	}
	return DefaultSocketPath
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
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("failed to create Unix socket listener: %w", err)
	}

	// Set appropriate permissions for the socket file
	if err := os.Chmod(path, 0660); err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to set socket permissions: %w", err)
	}

	return listener, nil
}

// createGRPCServer creates a gRPC server with appropriate configuration
func createGRPCServer() *grpc.Server {
	// Configure gRPC server with keepalive and timeouts
	kaep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second,
		PermitWithoutStream: true,
	}

	kasp := keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second,
		MaxConnectionAge:      30 * time.Second,
		MaxConnectionAgeGrace: 5 * time.Second,
		Time:                  5 * time.Second,
		Timeout:               1 * time.Second,
	}

	return grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
	)
}