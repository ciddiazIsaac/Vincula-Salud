package ports

import (
	"context"

	"github.com/minsal/vincula/internal/core/domain"
)

// EventPublisher define la interfaz para publicar eventos de dominio asincronamente.
type EventPublisher interface {
	// PublishClinicalEventRecorded publica un evento indicando que se ha registrado
	// un nuevo evento clinico para un paciente.
	PublishClinicalEventRecorded(ctx context.Context, event *domain.ClinicalEvent) error
}
