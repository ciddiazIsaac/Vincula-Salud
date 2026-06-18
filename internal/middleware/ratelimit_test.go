package middleware

import (
	"context"
	"testing"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRateLimitUnaryInterceptor(t *testing.T) {
	// 10 RPS, burst of 2
	rl := NewRateLimiter(rate.Limit(10), 2)
	interceptor := RateLimitUnaryInterceptor(rl)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/vinca.clinical.v1.ClinicalRecordService/GetPatientSummary"}

	ctx := context.WithValue(context.Background(), callerIdentityKey{}, CallerIdentity{CommonName: "client-1"})

	// First two should succeed (burst)
	_, err := interceptor(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	_, err = interceptor(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	// Third should fail due to rate limit
	_, err = interceptor(ctx, nil, info, handler)
	if err == nil {
		t.Fatal("expected rate limit error, got success")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.ResourceExhausted {
		t.Fatalf("expected ResourceExhausted, got %v", err)
	}
}
