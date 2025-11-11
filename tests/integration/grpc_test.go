package integration

import (
	"context"
	"net"
	"testing"

	"github.com/linuxsuren/api-testing/pkg/server"
	"github.com/linuxsuren/api-testing/pkg/testing/remote"
	grpcx "github.com/linuxsuren/atest-ext-ai/pkg/grpc"
	"github.com/linuxsuren/atest-ext-ai/pkg/plugin"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1 << 20

func startTestGRPCServer(t *testing.T) (*grpc.ClientConn, func()) {
	t.Helper()

	listener := bufconn.Listen(bufSize)
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcx.RequestIDInterceptor(),
			grpcx.LoggingInterceptor(),
		),
	)

	service, err := plugin.NewAIPluginService()
	require.NoError(t, err)
	remote.RegisterLoaderServer(server, service)

	go func() {
		if err := server.Serve(listener); err != nil {
			t.Logf("gRPC server stopped: %v", err)
		}
	}()

	conn, err := grpc.DialContext( //nolint:staticcheck
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	cleanup := func() {
		_ = conn.Close()
		server.Stop()
		if err := listener.Close(); err != nil {
			t.Logf("listener close error: %v", err)
		}
	}

	return conn, cleanup
}

func TestLoaderServerVersionOverGRPC(t *testing.T) {
	conn, cleanup := startTestGRPCServer(t)
	defer cleanup()

	client := remote.NewLoaderClient(conn)
	outgoingMD := metadata.Pairs("x-request-id", "integration-test")
	ctx := metadata.NewOutgoingContext(context.Background(), outgoingMD)

	var header metadata.MD
	version, err := client.GetVersion(ctx, &server.Empty{}, grpc.Header(&header))
	require.NoError(t, err)
	require.NotNil(t, version)
	require.Contains(t, version.Version, plugin.PluginVersion)

	headerIDs := header.Get("x-request-id")
	require.NotEmpty(t, headerIDs)
	require.Equal(t, "integration-test", headerIDs[0])
}

func TestLoaderServerMenusOverGRPC(t *testing.T) {
	conn, cleanup := startTestGRPCServer(t)
	defer cleanup()

	client := remote.NewLoaderClient(conn)
	menus, err := client.GetMenus(context.Background(), &server.Empty{})
	require.NoError(t, err)
	require.NotNil(t, menus)
	require.NotEmpty(t, menus.Data)
}
