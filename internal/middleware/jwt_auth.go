package middleware

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	JWTClaimsKey contextKey = "jwt_claims"
)

// JWTClaims holds the custom claims extracted from the JWT token.
type JWTClaims struct {
	UserID     string `json:"user_id"`
	HospitalID string `json:"hospital_id"`
	Role       string `json:"role"`
	jwt.RegisteredClaims
}

// JWTAuthUnaryInterceptor extracts and validates a JWT token from the authorization header.
func JWTAuthUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	newCtx, err := validateJWTFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid JWT: %v", err)
	}

	return handler(newCtx, req)
}

// JWTAuthStreamInterceptor extracts and validates a JWT token for stream requests.
func JWTAuthStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	newCtx, err := validateJWTFromContext(ss.Context())
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "invalid JWT: %v", err)
	}

	// Create a wrapped stream with the new context
	wrappedStream := &jwtWrappedServerStream{
		ServerStream: ss,
		ctx:          newCtx,
	}

	return handler(srv, wrappedStream)
}

func validateJWTFromContext(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, fmt.Errorf("missing metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return ctx, fmt.Errorf("missing authorization header")
	}

	tokenStr := authHeaders[0]
	if !strings.HasPrefix(strings.ToLower(tokenStr), "bearer ") {
		return ctx, fmt.Errorf("invalid authorization header format")
	}
	tokenStr = tokenStr[7:]

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default-secret-for-dev-only"
	}

	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return ctx, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		// Inject claims into context
		return context.WithValue(ctx, JWTClaimsKey, claims), nil
	}

	return ctx, fmt.Errorf("invalid token claims")
}

// ClaimsFromContext extracts the JWTClaims from the context, if present.
func ClaimsFromContext(ctx context.Context) (*JWTClaims, bool) {
	claims, ok := ctx.Value(JWTClaimsKey).(*JWTClaims)
	return claims, ok
}

// jwtWrappedServerStream wraps grpc.ServerStream to override its Context.
type jwtWrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *jwtWrappedServerStream) Context() context.Context {
	return w.ctx
}
