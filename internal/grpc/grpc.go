package grpc

import (
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/stormhead-org/backend/internal/jwt"
	"github.com/stormhead-org/backend/internal/middleware"
	"github.com/stormhead-org/backend/internal/orm"
	"github.com/stormhead-org/backend/internal/proto"

	authorizationgrpcpkg "github.com/stormhead-org/backend/internal/grpc/authorization"
	communitygrpcpkg "github.com/stormhead-org/backend/internal/grpc/community"
	postgrpcpkg "github.com/stormhead-org/backend/internal/grpc/post"
)

type GRPC struct {
	logger *zap.Logger
	host   string
	port   string
	server *grpc.Server
}

func NewGRPC(
	logger *zap.Logger,
	jwt *jwt.JWT,
	db *orm.PostgresClient,
	host string,
	port string,
	authServer *authorizationgrpcpkg.AuthorizationServer,
	communityServer *communitygrpcpkg.CommunityServer,
	postServer *postgrpcpkg.PostServer,
	commentServer *CommentServer,
	userServer *UserServer,
) (*GRPC, error) {
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(5, 600)
	authMiddleware := middleware.NewAuthorizationMiddleware(logger, jwt, db)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			rateLimitMiddleware,
			authMiddleware,
		),
	)

	// Register services
	proto.RegisterAuthorizationServiceServer(grpcServer, authServer)
	proto.RegisterCommunityServiceServer(grpcServer, communityServer)
	proto.RegisterPostServiceServer(grpcServer, postServer)
	proto.RegisterCommentServiceServer(grpcServer, commentServer)
	proto.RegisterUserServiceServer(grpcServer, userServer)

	// Search API
	// searchServer := NewSearchServer(logger, database, broker)
	// protopkg.RegisterSearchServiceServer(grpcServer, searchServer)

	// Health API
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	// Reflection API
	reflection.Register(grpcServer)

	return &GRPC{
		logger: logger,
		host:   host,
		port:   port,
		server: grpcServer,
	}, nil
}

func (this *GRPC) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", this.host, this.port))
	if err != nil {
		return err
	}

	go func() {
		this.logger.Info("GRPC server started", zap.String("addr", listener.Addr().String()))
		err := this.server.Serve(listener)
		if err != nil {
			this.logger.Error("GRPC server stopped", zap.Error(err))
		}
	}()

	return nil
}

func (this *GRPC) Stop() error {
	this.server.GracefulStop()
	this.logger.Info("GRPC server stopped gracefully")
	return nil
}
