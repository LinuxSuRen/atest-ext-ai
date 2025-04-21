package main

import (
	"context"
	"fmt"

	"github.com/linuxsuren/api-testing/pkg/server"
	testing "github.com/linuxsuren/api-testing/pkg/testing"
	"github.com/linuxsuren/api-testing/pkg/testing/remote"
	grpc "google.golang.org/grpc"
)

func query(ctx context.Context, store testing.Store, sql string) (data *server.DataQueryResult, err error) {
	address := store.Kind.URL
	var conn *grpc.ClientConn
	if conn, err = grpc.Dial(address, grpc.WithInsecure()); err == nil {
		ctx = remote.WithStoreContext(ctx, &store)
		writer := &gRPCLoader{
			store:  &store,
			ctx:    ctx,
			client: remote.NewLoaderClient(conn),
			conn:   conn,
		}

		data, err = writer.client.Query(ctx, &server.DataQuery{
			Sql: sql,
		})
	} else {
		err = fmt.Errorf("failed to connect: %s, %v", address, err)
	}
	return
}

type gRPCLoader struct {
	store  *testing.Store
	client remote.LoaderClient
	ctx    context.Context
	conn   *grpc.ClientConn
}
