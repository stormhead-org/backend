package usergrpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *UserServer) Heartbeat(ctx context.Context, request *protopkg.HeartbeatRequest) (*protopkg.HeartbeatResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Heartbeat not implemented")
}
