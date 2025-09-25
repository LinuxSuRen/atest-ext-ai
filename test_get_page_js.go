package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/linuxsuren/api-testing/pkg/server"
	"github.com/linuxsuren/api-testing/pkg/testing/remote"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to the AI plugin socket
	conn, err := grpc.Dial("unix:///tmp/atest-ext-ai.sock",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", "/tmp/atest-ext-ai.sock", timeout)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to connect to AI plugin: %v", err)
	}
	defer conn.Close()

	client := remote.NewLoaderClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test GetMenus call
	fmt.Println("=== Testing GetMenus ===")
	menus, err := client.GetMenus(ctx, &server.Empty{})
	if err != nil {
		log.Printf("GetMenus failed: %v", err)
	} else {
		fmt.Printf("GetMenus success: %+v\n", menus)
		for i, menu := range menus.Data {
			fmt.Printf("Menu %d: name=%s, index=%s, icon=%s\n", i, menu.Name, menu.Index, menu.Icon)
		}
	}

	// Test GetPageOfJS call
	fmt.Println("\n=== Testing GetPageOfJS ===")
	jsResult, err := client.GetPageOfJS(ctx, &server.SimpleName{Name: "ai-chat"})
	if err != nil {
		log.Printf("GetPageOfJS failed: %v", err)
	} else {
		fmt.Printf("GetPageOfJS success: %t\n", jsResult.Success)
		if jsResult.Success {
			fmt.Printf("JavaScript code length: %d characters\n", len(jsResult.Message))
			fmt.Printf("JavaScript code preview (first 200 chars):\n%s...\n",
				jsResult.Message[:min(200, len(jsResult.Message))])
		} else {
			fmt.Printf("Error message: %s\n", jsResult.Message)
		}
	}

	// Test GetPageOfCSS call
	fmt.Println("\n=== Testing GetPageOfCSS ===")
	cssResult, err := client.GetPageOfCSS(ctx, &server.SimpleName{Name: "ai-chat"})
	if err != nil {
		log.Printf("GetPageOfCSS failed: %v", err)
	} else {
		fmt.Printf("GetPageOfCSS success: %t\n", cssResult.Success)
		if cssResult.Success {
			fmt.Printf("CSS code length: %d characters\n", len(cssResult.Message))
		} else {
			fmt.Printf("Error message: %s\n", cssResult.Message)
		}
	}

	// Test Verify call
	fmt.Println("\n=== Testing Verify ===")
	status, err := client.Verify(ctx, &server.Empty{})
	if err != nil {
		log.Printf("Verify failed: %v", err)
	} else {
		fmt.Printf("Verify success: ready=%t, version=%s, message=%s\n",
			status.Ready, status.Version, status.Message)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}