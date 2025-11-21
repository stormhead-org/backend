package grpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	protopkg "github.com/stormhead-org/backend/internal/proto"
)

func (s *CommentServer) Stream(request *protopkg.StreamCommentRequest, stream protopkg.CommentService_StreamServer) error {
	return status.Errorf(codes.Unimplemented, "method Stream not implemented")
}
