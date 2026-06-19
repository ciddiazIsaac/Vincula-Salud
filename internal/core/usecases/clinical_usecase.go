package usecases

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/minsal/vincula/internal/core/domain"
	"github.com/minsal/vincula/internal/core/ports"
)

type clinicalUseCase struct {
	repo      ports.ClinicalRecordRepository
	publisher ports.EventPublisher
}

func NewClinicalUseCase(repo ports.ClinicalRecordRepository, publisher ports.EventPublisher) ports.ClinicalRecordUseCase {
	return &clinicalUseCase{
		repo:      repo,
		publisher: publisher,
	}
}

func (uc *clinicalUseCase) GetPatientSummary(ctx context.Context, patientRun string) (*domain.PatientSummary, error) {
	events, err := uc.repo.GetRecentEvents(ctx, patientRun, 10)
	if err != nil {
		return nil, err
	}

	summary := &domain.PatientSummary{
		PatientRun: patientRun,
		LastUpdate: time.Now(),
	}

	var maxUpdate time.Time

	for _, ev := range events {
		var data map[string]interface{}
		if err := json.Unmarshal(ev.EventDataJSON, &data); err != nil {
			slog.WarnContext(ctx, "Failed to unmarshal event data", "error", err, "patient_run", patientRun)
			continue
		}

		switch ev.EventType {
		case "allergy":
			if val, ok := data["alergia"]; ok {
				if strVal, ok := val.(string); ok {
					summary.ActiveAllergies = append(summary.ActiveAllergies, strVal)
				}
			}
		case "diagnosis":
			if val, ok := data["diagnostico"]; ok {
				if strVal, ok := val.(string); ok {
					summary.ActiveDiagnoses = append(summary.ActiveDiagnoses, strVal)
				}
			}
		case "prescription":
			if val, ok := data["medicamento"]; ok {
				if strVal, ok := val.(string); ok {
					summary.ActiveMedications = append(summary.ActiveMedications, strVal)
				}
			}
		}

		if ev.EventTimestamp.After(maxUpdate) {
			maxUpdate = ev.EventTimestamp
		}
	}

	if !maxUpdate.IsZero() {
		summary.LastUpdate = maxUpdate
	}

	return summary, nil
}

func (uc *clinicalUseCase) RecordClinicalEvent(ctx context.Context, event *domain.ClinicalEvent) (*domain.ClinicalEvent, error) {
	event.EventID = uuid.New().String()
	event.RecordedAt = time.Now()
	if event.EventTimestamp.IsZero() {
		event.EventTimestamp = time.Now()
	}

	if err := uc.repo.SaveEvent(ctx, event); err != nil {
		return nil, err
	}

	// Emitir evento asíncrono si el publisher está disponible
	if uc.publisher != nil {
		if err := uc.publisher.PublishClinicalEventRecorded(ctx, event); err != nil {
			// Solo logueamos el error, no fallamos la operación principal (Eventual Consistency)
			slog.ErrorContext(ctx, "Failed to publish clinical event", "error", err, "event_id", event.EventID)
		} else {
			slog.InfoContext(ctx, "Clinical event published successfully", "event_id", event.EventID)
		}
	}

	return event, nil
}

func (uc *clinicalUseCase) ListClinicalEvents(ctx context.Context, patientRun string, eventTypeFilter string, pageSize int) ([]*domain.ClinicalEvent, error) {
	if pageSize == 0 {
		pageSize = 100
	}
	return uc.repo.ListEvents(ctx, patientRun, eventTypeFilter, pageSize)
}

func (uc *clinicalUseCase) RevokeConsent(ctx context.Context, patientRun string, dataCategory string) error {
	return uc.repo.RevokeConsent(ctx, patientRun, dataCategory)
}
