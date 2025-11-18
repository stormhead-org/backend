package middleware

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimiterInterceptor provides a gRPC unary interceptor for rate limiting.
type RateLimiterInterceptor struct {
	mu        sync.Mutex
	limiters  map[string]*rate.Limiter
	rps       rate.Limit
	burst     int
}

// NewRateLimiterInterceptor creates a new rate limiter interceptor.
func NewRateLimiterInterceptor(rps float64, burst int) *RateLimiterInterceptor {
	return &RateLimiterInterceptor{
		limiters:  make(map[string]*rate.Limiter),
		rps:       rate.Limit(rps),
		burst:     burst,
	}
}

// Unary returns a gRPC unary server interceptor that performs rate limiting.
func (i *RateLimiterInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		p, ok := peer.FromContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Internal, "could not get peer from context")
		}

		// Use the IP address as the key.
		ip := p.Addr.String()

		i.mu.Lock()
		limiter, exists := i.limiters[ip]
		if !exists {
			limiter = rate.NewLimiter(i.rps, i.burst)
			i.limiters[ip] = limiter
		}
		i.mu.Unlock()

		if !limiter.Allow() {
			return nil, status.Errorf(codes.ResourceExhausted, "too many requests")
		}

		return handler(ctx, req)
	}
}
