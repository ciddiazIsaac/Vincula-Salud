package middleware

import (
	"strings"
	"testing"

	clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
)

func TestValidateRecordClinicalEvent(t *testing.T) {
	tests := []struct {
		name    string
		req     *clinicalv1.RecordClinicalEventRequest
		wantErr bool
		errContains string
	}{
		{
			name: "valid request",
			req: &clinicalv1.RecordClinicalEventRequest{
				PatientRun:       "12345678-9",
				EventType:        "allergy",
				EventDataJson:    []byte(`{"alergia":"penicilina"}`),
				AuthorCredential: "DR-TEST",
			},
			wantErr: false,
		},
		{
			name: "valid run with dots",
			req: &clinicalv1.RecordClinicalEventRequest{
				PatientRun:       "12.345.678-9",
				EventType:        "diagnosis",
				EventDataJson:    []byte(`{"diag":"covid"}`),
				AuthorCredential: "DR-TEST",
			},
			wantErr: false,
		},
		{
			name: "invalid run format",
			req: &clinicalv1.RecordClinicalEventRequest{
				PatientRun:       "1234567-",
				EventType:        "allergy",
				EventDataJson:    []byte(`{}`),
				AuthorCredential: "DR-TEST",
			},
			wantErr: true,
			errContains: "valid RUN format",
		},
		{
			name: "invalid event type",
			req: &clinicalv1.RecordClinicalEventRequest{
				PatientRun:       "12345678-9",
				EventType:        "unknown_type",
				EventDataJson:    []byte(`{}`),
				AuthorCredential: "DR-TEST",
			},
			wantErr: true,
			errContains: "not a valid type",
		},
		{
			name: "missing author credential",
			req: &clinicalv1.RecordClinicalEventRequest{
				PatientRun:       "12345678-9",
				EventType:        "allergy",
				EventDataJson:    []byte(`{}`),
			},
			wantErr: true,
			errContains: "author_credential is required",
		},
		{
			name: "invalid json utf8",
			req: &clinicalv1.RecordClinicalEventRequest{
				PatientRun:       "12345678-9",
				EventType:        "allergy",
				EventDataJson:    []byte{0xff, 0xfe, 0xfd}, // invalid utf8
				AuthorCredential: "DR-TEST",
			},
			wantErr: true,
			errContains: "valid UTF-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRecordClinicalEvent(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRecordClinicalEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}
