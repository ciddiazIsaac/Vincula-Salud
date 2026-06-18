package ports

import (
	"context"

	"github.com/minsal/vincula/internal/core/domain"
)

type ClinicalRecordUseCase interface {
	GetPatientSummary(ctx context.Context, patientRun string) (*domain.PatientSummary, error)
	RecordClinicalEvent(ctx context.Context, event *domain.ClinicalEvent) (*domain.ClinicalEvent, error)
	ListClinicalEvents(ctx context.Context, patientRun string, eventTypeFilter string, pageSize int) ([]*domain.ClinicalEvent, error)
	RevokeConsent(ctx context.Context, patientRun string, dataCategory string) error
}
