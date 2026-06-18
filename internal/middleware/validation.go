package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"unicode/utf8"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
)

const (
	// maxEventDataSize is the maximum allowed size for event_data_json (1 MB).
	maxEventDataSize = 1 * 1024 * 1024
	// maxPageSize is the maximum allowed page size for list operations.
	maxPageSize = 1000
	// defaultPageSize is the default page size when none is specified.
	defaultPageSize = 100
)

// validRunRegex matches a Chilean RUN in the format "12345678-9" or "12.345.678-K".
// Accepts RUN with or without dots.
var validRunRegex = regexp.MustCompile(`^(\d{1,2}\.?\d{3}\.?\d{3})-[\dkK]$`)

// allowedEventTypes is the set of permitted event types.
var allowedEventTypes = map[string]bool{
	"allergy":      true,
	"diagnosis":    true,
	"prescription": true,
	"lab_result":   true,
	"procedure":    true,
	"consultation": true,
}

// ValidationUnaryInterceptor is a gRPC unary server interceptor that validates
// request fields before they reach the handler. Rejects malformed requests
// with codes.InvalidArgument.
func ValidationUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if err := validateRequest(req); err != nil {
		slog.WarnContext(ctx, "Request validation failed",
			"method", info.FullMethod,
			"error", err,
		)
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}
	return handler(ctx, req)
}

// validateRequest dispatches validation based on the request type.
func validateRequest(req interface{}) error {
	switch r := req.(type) {
	case *clinicalv1.GetPatientSummaryRequest:
		return validateGetPatientSummary(r)
	case *clinicalv1.RecordClinicalEventRequest:
		return validateRecordClinicalEvent(r)
	case *clinicalv1.ListClinicalEventsRequest:
		return validateListClinicalEvents(r)
	case *clinicalv1.RevokeConsentRequest:
		return validateRevokeConsent(r)
	default:
		return nil // Unknown request types pass through
	}
}

func validateGetPatientSummary(req *clinicalv1.GetPatientSummaryRequest) error {
	if req.PatientRun == "" {
		return fmt.Errorf("patient_run is required")
	}
	if !validRunRegex.MatchString(req.PatientRun) {
		return fmt.Errorf("patient_run '%s' does not match valid RUN format (e.g., 12345678-9)", req.PatientRun)
	}
	return nil
}

func validateRecordClinicalEvent(req *clinicalv1.RecordClinicalEventRequest) error {
	if req.PatientRun == "" {
		return fmt.Errorf("patient_run is required")
	}
	if !validRunRegex.MatchString(req.PatientRun) {
		return fmt.Errorf("patient_run '%s' does not match valid RUN format", req.PatientRun)
	}
	if req.EventType == "" {
		return fmt.Errorf("event_type is required")
	}
	if !allowedEventTypes[req.EventType] {
		return fmt.Errorf("event_type '%s' is not a valid type; allowed: allergy, diagnosis, prescription, lab_result, procedure, consultation", req.EventType)
	}
	if len(req.EventDataJson) == 0 {
		return fmt.Errorf("event_data_json is required")
	}
	if len(req.EventDataJson) > maxEventDataSize {
		return fmt.Errorf("event_data_json exceeds maximum size of %d bytes", maxEventDataSize)
	}
	if !utf8.Valid(req.EventDataJson) {
		return fmt.Errorf("event_data_json must be valid UTF-8 encoded JSON")
	}
	if req.AuthorCredential == "" {
		return fmt.Errorf("author_credential is required")
	}
	return nil
}

func validateListClinicalEvents(req *clinicalv1.ListClinicalEventsRequest) error {
	if req.PatientRun == "" {
		return fmt.Errorf("patient_run is required")
	}
	if !validRunRegex.MatchString(req.PatientRun) {
		return fmt.Errorf("patient_run '%s' does not match valid RUN format", req.PatientRun)
	}
	if req.EventTypeFilter != "" && !allowedEventTypes[req.EventTypeFilter] {
		return fmt.Errorf("event_type_filter '%s' is not a valid type", req.EventTypeFilter)
	}
	if req.PageSize < 0 {
		return fmt.Errorf("page_size must be non-negative")
	}
	if req.PageSize > maxPageSize {
		return fmt.Errorf("page_size cannot exceed %d", maxPageSize)
	}
	return nil
}

func validateRevokeConsent(req *clinicalv1.RevokeConsentRequest) error {
	if req.PatientRun == "" {
		return fmt.Errorf("patient_run is required")
	}
	if !validRunRegex.MatchString(req.PatientRun) {
		return fmt.Errorf("patient_run '%s' does not match valid RUN format", req.PatientRun)
	}
	if req.DataCategory == "" {
		return fmt.Errorf("data_category is required")
	}
	return nil
}
