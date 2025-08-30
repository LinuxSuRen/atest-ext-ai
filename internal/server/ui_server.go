//go:build embed
// +build embed

package server

import (
	"context"
	"embed"
	"io/fs"

	"atest-ext-ai-core/internal/logger"
	pb "github.com/linuxsuren/api-testing/pkg/server"
	"google.golang.org/grpc"
)

//go:embed assets/*
var embeddedAssets embed.FS

// TODO: Embed frontend static files when build process is ready
// For now, we'll serve placeholder content

// UIExtensionServer implements the gRPC UI extension interface
type UIExtensionServer struct {
	pb.UnimplementedUIExtensionServer
	// Embedded file system for frontend assets
	assets embed.FS
}

// NewUIExtensionServer creates a new UI extension server instance
func NewUIExtensionServer() *UIExtensionServer {
	return &UIExtensionServer{
		assets: embeddedAssets,
	}
}

// GetJSContent returns JavaScript content for the AI plugin
func (s *UIExtensionServer) GetJSContent() string {
	logger.Debug("Serving AI plugin JavaScript content")
	return s.getJSContent()
}

// GetCSSContent returns CSS content for the AI plugin
func (s *UIExtensionServer) GetCSSContent() string {
	logger.Debug("Serving AI plugin CSS content")
	return s.getCSSContent()
}

// getJSContent returns JavaScript content for the AI plugin
func (s *UIExtensionServer) getJSContent() string {
	// Try to read from embedded assets first
	if jsContent, err := s.readEmbeddedFile("assets/index.js"); err == nil {
		return jsContent
	}

	// Fallback to placeholder content
	jsContent := `
		// AI Plugin Frontend JavaScript
		console.log('AI Plugin UI loaded');
		
		// Initialize AI Plugin UI
		if (typeof window !== 'undefined') {
			window.AIPlugin = {
				init: function() {
					console.log('AI Plugin initialized');
					// Add AI plugin functionality here
				}
			};
			
			// Auto-initialize when DOM is ready
			if (document.readyState === 'loading') {
				document.addEventListener('DOMContentLoaded', window.AIPlugin.init);
			} else {
				window.AIPlugin.init();
			}
		}
	`

	return jsContent
}

// GetMenus implements the UIExtensionServer interface
func (s *UIExtensionServer) GetMenus(ctx context.Context, req *pb.Empty) (*pb.MenuList, error) {
	logger.Debug("Getting AI plugin menus")
	return &pb.MenuList{
		Data: []*pb.Menu{
			{
				Name:    "AI Assistant",
				Index:   "/ai-plugin",
				Icon:    "robot",
				Version: 1,
			},
		},
	}, nil
}

// GetPageOfJS implements the UIExtensionServer interface
func (s *UIExtensionServer) GetPageOfJS(ctx context.Context, req *pb.SimpleName) (*pb.CommonResult, error) {
	logger.Debug("Serving AI plugin JS content")
	jsContent := s.getJSContent()
	return &pb.CommonResult{
		Success: true,
		Message: jsContent,
	}, nil
}

// GetPageOfCSS implements the UIExtensionServer interface
func (s *UIExtensionServer) GetPageOfCSS(ctx context.Context, req *pb.SimpleName) (*pb.CommonResult, error) {
	logger.Debug("Serving AI plugin CSS content")
	cssContent := s.getCSSContent()
	return &pb.CommonResult{
		Success: true,
		Message: cssContent,
	}, nil
}

// getCSSContent returns CSS content for the AI plugin
func (s *UIExtensionServer) getCSSContent() string {
	// Try to read from embedded assets first
	if cssContent, err := s.readEmbeddedFile("assets/style.css"); err == nil {
		return cssContent
	}

	// Fallback to placeholder content
	cssContent := `
		/* AI Plugin Frontend CSS */
		.ai-plugin-container {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
			padding: 16px;
			max-width: 1200px;
			margin: 0 auto;
		}
		
		.ai-chat-container {
			border: 1px solid #e1e5e9;
			border-radius: 8px;
			padding: 16px;
			margin-bottom: 16px;
			background: #ffffff;
		}
		
		.sql-editor-container {
			border: 1px solid #e1e5e9;
			border-radius: 8px;
			padding: 16px;
			margin-bottom: 16px;
			background: #f8f9fa;
		}
		
		.result-display-container {
			border: 1px solid #e1e5e9;
			border-radius: 8px;
			padding: 16px;
			background: #ffffff;
		}
	`

	return cssContent
}

// readEmbeddedFile reads a file from the embedded file system
func (s *UIExtensionServer) readEmbeddedFile(path string) (string, error) {
	data, err := fs.ReadFile(s.assets, path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// getHTMLContent returns the main HTML content for the AI plugin
func (s *UIExtensionServer) getHTMLContent() string {
	// Try to read from embedded assets first
	if htmlContent, err := s.readEmbeddedFile("assets/index.html"); err == nil {
		return htmlContent
	}

	// Fallback to placeholder content
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AI Plugin</title>
</head>
<body>
    <div id="app">
        <div class="ai-plugin-container">
            <h1>AI Assistant Plugin</h1>
            <div id="ai-chat-container"></div>
        </div>
    </div>
</body>
</html>`
}

// RegisterUIExtensionServer registers the UI extension server with gRPC
func RegisterUIExtensionServer(s *grpc.Server, srv *UIExtensionServer) {
	pb.RegisterUIExtensionServer(s, srv)
	logger.Info("UI Extension Server registered successfully")
}
