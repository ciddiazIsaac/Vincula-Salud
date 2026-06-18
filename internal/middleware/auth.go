package middleware

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// CallerIdentity represents the authenticated identity of a gRPC caller,
// extracted from the mTLS client certificate.
type CallerIdentity struct {
	// CommonName is the CN field from the client certificate's Subject.
	CommonName string
	// DNSNames are the Subject Alternative Names (SANs) from the certificate.
	DNSNames []string
	// Organization is the O field from the client certificate's Subject.
	Organization []string
	// SerialNumber is the serial number of the client certificate.
	SerialNumber string
}

type callerIdentityKey struct{}

// IdentityFromContext retrieves the CallerIdentity from the context.
// Returns the identity and true if found, zero value and false otherwise.
func IdentityFromContext(ctx context.Context) (CallerIdentity, bool) {
	id, ok := ctx.Value(callerIdentityKey{}).(CallerIdentity)
	return id, ok
}

// AuthUnaryInterceptor is a gRPC unary server interceptor that authenticates
// the caller by extracting identity information from the mTLS client certificate.
// It rejects requests that do not present a valid client certificate.
func AuthUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	identity, err := extractIdentity(ctx)
	if err != nil {
		slog.WarnContext(ctx, "Authentication failed",
			"method", info.FullMethod,
			"error", err,
		)
		return nil, err
	}

	slog.DebugContext(ctx, "Caller authenticated",
		"method", info.FullMethod,
		"cn", identity.CommonName,
		"org", identity.Organization,
	)

	ctx = context.WithValue(ctx, callerIdentityKey{}, identity)
	return handler(ctx, req)
}

// AuthStreamInterceptor is a gRPC stream server interceptor that authenticates
// the caller by extracting identity information from the mTLS client certificate.
func AuthStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	identity, err := extractIdentity(ss.Context())
	if err != nil {
		slog.WarnContext(ss.Context(), "Stream authentication failed",
			"method", info.FullMethod,
			"error", err,
		)
		return err
	}

	ctx := context.WithValue(ss.Context(), callerIdentityKey{}, identity)
	wrapped := &wrappedServerStream{ServerStream: ss, ctx: ctx}

	slog.DebugContext(ctx, "Stream caller authenticated",
		"method", info.FullMethod,
		"cn", identity.CommonName,
	)

	return handler(srv, wrapped)
}

// extractIdentity extracts the caller identity from the TLS peer certificate.
func extractIdentity(ctx context.Context) (CallerIdentity, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return CallerIdentity{}, status.Error(codes.Unauthenticated, "no peer information available")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return CallerIdentity{}, status.Error(codes.Unauthenticated, "connection is not using TLS")
	}

	if len(tlsInfo.State.PeerCertificates) == 0 {
		return CallerIdentity{}, status.Error(codes.Unauthenticated, "no client certificate presented")
	}

	cert := tlsInfo.State.PeerCertificates[0]
	return CallerIdentity{
		CommonName:   cert.Subject.CommonName,
		DNSNames:     cert.DNSNames,
		Organization: cert.Subject.Organization,
		SerialNumber: cert.SerialNumber.String(),
	}, nil
}

// wrappedServerStream wraps a grpc.ServerStream to override the Context().
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
