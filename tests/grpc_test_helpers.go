package tests

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// NewBufConnGRPCServer creates a gRPC server that listens on a bufconn.Listener.
// It returns the server and a client connection to it.
func NewBufConnGRPCServer(ctx context.Context, registerServer func(s *grpc.Server)) (*grpc.Server, *grpc.ClientConn, error) {
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()

	registerServer(s)

	go func() {
		if err := s.Serve(lis); err != nil {
			zap.L().Error("gRPC server exited with error", zap.Error(err))
		}
	}()

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial bufnet: %w", err)
	}

	return s, conn, nil
}
