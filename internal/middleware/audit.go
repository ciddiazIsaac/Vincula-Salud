package middleware

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuditUnaryInterceptor is a gRPC unary server interceptor that logs an
// immutable audit record for every RPC call. This is essential for clinical
// data compliance (ley 20.584 / HIPAA).
//
// Each audit entry includes: caller identity, method, timestamp, duration,
// result status, and relevant request fields (e.g., patient_run).
func AuditUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	// Extract caller identity (set by AuthUnaryInterceptor)
	identity, hasIdentity := IdentityFromContext(ctx)
	callerCN := "unknown"
	callerOrg := "unknown"
	if hasIdentity {
		callerCN = identity.CommonName
		if len(identity.Organization) > 0 {
			callerOrg = identity.Organization[0]
		}
	}

	// Extract patient_run from the request if available
	patientRun := extractPatientRun(req)

	// Execute the handler
	resp, err := handler(ctx, req)

	duration := time.Since(start)
	resultCode := codes.OK
	if err != nil {
		resultCode = status.Code(err)
	}

	// Emit structured audit log entry
	slog.InfoContext(ctx, "AUDIT",
		"caller_cn", callerCN,
		"caller_org", callerOrg,
		"method", info.FullMethod,
		"patient_run", patientRun,
		"result_code", resultCode.String(),
		"duration_ms", duration.Milliseconds(),
		"timestamp_utc", start.UTC().Format(time.RFC3339Nano),
	)

	return resp, err
}

// patientRunExtractor is an interface for requests that contain a patient_run field.
type patientRunExtractor interface {
	GetPatientRun() string
}

// extractPatientRun safely extracts the patient_run field from known request types.
func extractPatientRun(req interface{}) string {
	if extractor, ok := req.(patientRunExtractor); ok {
		return extractor.GetPatientRun()
	}
	return ""
}
