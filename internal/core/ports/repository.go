package ports

import (
	"context"

	"github.com/minsal/vincula/internal/core/domain"
)

type ClinicalRecordRepository interface {
	GetRecentEvents(ctx context.Context, patientRun string, limit int) ([]*domain.ClinicalEvent, error)
	ListEvents(ctx context.Context, patientRun string, eventTypeFilter string, limit int) ([]*domain.ClinicalEvent, error)
	SaveEvent(ctx context.Context, event *domain.ClinicalEvent) error
	RevokeConsent(ctx context.Context, patientRun string, dataCategory string) error
}
