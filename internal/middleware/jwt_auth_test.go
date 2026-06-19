package middleware

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testSecret = "test-secret-key-for-jwt-tests"

// helper: generate a signed JWT token string with the given claims and secret.
func generateTestToken(t *testing.T, claims JWTClaims, secret string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return tokenStr
}

// helper: create a context with gRPC incoming metadata containing the given authorization value.
func ctxWithAuth(authValue string) context.Context {
	md := metadata.Pairs("authorization", authValue)
	return metadata.NewIncomingContext(context.Background(), md)
}

// ---------------------------------------------------------------------------
// validateJWTFromContext tests
// ---------------------------------------------------------------------------

func TestValidateJWT_MissingMetadata(t *testing.T) {
	// Context without any gRPC metadata.
	_, err := validateJWTFromContext(context.Background())
	if err == nil {
		t.Fatal("expected error for missing metadata")
	}
}

func TestValidateJWT_MissingAuthHeader(t *testing.T) {
	// Metadata present but no "authorization" key.
	md := metadata.Pairs("other-key", "value")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := validateJWTFromContext(ctx)
	if err == nil {
		t.Fatal("expected error for missing authorization header")
	}
}

func TestValidateJWT_InvalidBearerFormat(t *testing.T) {
	ctx := ctxWithAuth("Basic dXNlcjpwYXNz")

	_, err := validateJWTFromContext(ctx)
	if err == nil {
		t.Fatal("expected error for non-Bearer authorization")
	}
}

func TestValidateJWT_InvalidToken(t *testing.T) {
	ctx := ctxWithAuth("Bearer not-a-real-jwt-token")

	_, err := validateJWTFromContext(ctx)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	claims := JWTClaims{
		UserID:     "user-1",
		HospitalID: "hosp-1",
		Role:       "doctor",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	tokenStr := generateTestToken(t, claims, testSecret)
	ctx := ctxWithAuth("Bearer " + tokenStr)

	_, err := validateJWTFromContext(ctx)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateJWT_WrongSigningMethod(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	// Create a token with "none" signing method (alg: none).
	token := jwt.NewWithClaims(jwt.SigningMethodNone, &JWTClaims{
		UserID: "user-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	ctx := ctxWithAuth("Bearer " + tokenStr)

	_, err = validateJWTFromContext(ctx)
	if err == nil {
		t.Fatal("expected error for non-HMAC signing method")
	}
}

func TestValidateJWT_ValidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	claims := JWTClaims{
		UserID:     "user-42",
		HospitalID: "hosp-central",
		Role:       "doctor",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tokenStr := generateTestToken(t, claims, testSecret)
	ctx := ctxWithAuth("Bearer " + tokenStr)

	newCtx, err := validateJWTFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error for valid token: %v", err)
	}

	extracted, ok := ClaimsFromContext(newCtx)
	if !ok {
		t.Fatal("expected claims in context")
	}
	if extracted.UserID != "user-42" {
		t.Errorf("UserID = %q, want %q", extracted.UserID, "user-42")
	}
	if extracted.HospitalID != "hosp-central" {
		t.Errorf("HospitalID = %q, want %q", extracted.HospitalID, "hosp-central")
	}
	if extracted.Role != "doctor" {
		t.Errorf("Role = %q, want %q", extracted.Role, "doctor")
	}
}

func TestValidateJWT_CustomSecret(t *testing.T) {
	customSecret := "my-custom-production-secret"
	t.Setenv("JWT_SECRET", customSecret)

	claims := JWTClaims{
		UserID: "user-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	tokenStr := generateTestToken(t, claims, customSecret)
	ctx := ctxWithAuth("Bearer " + tokenStr)

	_, err := validateJWTFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error with custom secret: %v", err)
	}

	// Token signed with wrong secret should fail.
	wrongToken := generateTestToken(t, claims, "wrong-secret")
	ctx2 := ctxWithAuth("Bearer " + wrongToken)

	_, err = validateJWTFromContext(ctx2)
	if err == nil {
		t.Fatal("expected error for token signed with wrong secret")
	}
}

func TestValidateJWT_DefaultSecret(t *testing.T) {
	// Ensure JWT_SECRET is empty to trigger the default.
	os.Unsetenv("JWT_SECRET")

	claims := JWTClaims{
		UserID: "user-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	tokenStr := generateTestToken(t, claims, "default-secret-for-dev-only")
	ctx := ctxWithAuth("Bearer " + tokenStr)

	_, err := validateJWTFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error with default secret: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ClaimsFromContext tests
// ---------------------------------------------------------------------------

func TestClaimsFromContext_Present(t *testing.T) {
	claims := &JWTClaims{UserID: "user-1", HospitalID: "hosp-1", Role: "admin"}
	ctx := context.WithValue(context.Background(), JWTClaimsKey, claims)

	extracted, ok := ClaimsFromContext(ctx)
	if !ok {
		t.Fatal("expected claims to be present")
	}
	if extracted.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", extracted.UserID, "user-1")
	}
}

func TestClaimsFromContext_Missing(t *testing.T) {
	_, ok := ClaimsFromContext(context.Background())
	if ok {
		t.Fatal("expected claims to be missing")
	}
}

// ---------------------------------------------------------------------------
// JWTAuthUnaryInterceptor tests
// ---------------------------------------------------------------------------

func TestJWTAuthUnaryInterceptor_ValidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	claims := JWTClaims{
		UserID:     "user-99",
		HospitalID: "hosp-test",
		Role:       "nurse",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	tokenStr := generateTestToken(t, claims, testSecret)
	ctx := ctxWithAuth("Bearer " + tokenStr)

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		// Verify claims were injected into context.
		extracted, ok := ClaimsFromContext(ctx)
		if !ok {
			t.Error("expected claims in handler context")
		}
		if extracted.UserID != "user-99" {
			t.Errorf("UserID in handler = %q, want %q", extracted.UserID, "user-99")
		}
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	resp, err := JWTAuthUnaryInterceptor(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Error("expected handler to be called")
	}
	if resp != "ok" {
		t.Errorf("unexpected response: %v", resp)
	}
}

func TestJWTAuthUnaryInterceptor_InvalidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	ctx := ctxWithAuth("Bearer invalid-token")

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("handler should not be called for invalid token")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := JWTAuthUnaryInterceptor(ctx, nil, info, handler)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated code, got %v", st.Code())
	}
}

func TestJWTAuthUnaryInterceptor_MissingToken(t *testing.T) {
	// Context with metadata but no authorization header.
	md := metadata.Pairs("other", "value")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("handler should not be called")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := JWTAuthUnaryInterceptor(ctx, nil, info, handler)
	if err == nil {
		t.Fatal("expected error for missing token")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated code, got %v", st.Code())
	}
}
