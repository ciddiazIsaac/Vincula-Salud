package middleware

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		limiter, exists = rl.limiters[key]
		if !exists {
			limiter = rate.NewLimiter(rl.r, rl.b)
			rl.limiters[key] = limiter
		}
		rl.mu.Unlock()
	}

	return limiter
}

func RateLimitUnaryInterceptor(rl *RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		identity, ok := IdentityFromContext(ctx)
		key := "anonymous"
		if ok {
			key = identity.CommonName
		}

		limiter := rl.getLimiter(key)
		if !limiter.Allow() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}

func RateLimitStreamInterceptor(rl *RateLimiter) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		identity, ok := IdentityFromContext(ss.Context())
		key := "anonymous"
		if ok {
			key = identity.CommonName
		}

		limiter := rl.getLimiter(key)
		if !limiter.Allow() {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(srv, ss)
	}
}
