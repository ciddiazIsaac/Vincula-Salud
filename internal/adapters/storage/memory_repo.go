package storage

import (
	"context"
	"sort"
	"sync"

	"github.com/minsal/vincula/internal/core/domain"
	"github.com/minsal/vincula/internal/core/ports"
)

type InMemoryClinicalRepo struct {
	mu     sync.RWMutex
	events map[string][]*domain.ClinicalEvent
}

func NewInMemoryClinicalRepo() ports.ClinicalRecordRepository {
	return &InMemoryClinicalRepo{
		events: make(map[string][]*domain.ClinicalEvent),
	}
}

func (r *InMemoryClinicalRepo) GetRecentEvents(ctx context.Context, patientRun string, limit int) ([]*domain.ClinicalEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := r.events[patientRun]
	sorted := make([]*domain.ClinicalEvent, len(all))
	copy(sorted, all)

	// Sort descending by event timestamp
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].EventTimestamp.After(sorted[j].EventTimestamp)
	})

	if len(sorted) > limit {
		return sorted[:limit], nil
	}
	return sorted, nil
}

func (r *InMemoryClinicalRepo) ListEvents(ctx context.Context, patientRun string, eventTypeFilter string, limit int) ([]*domain.ClinicalEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := r.events[patientRun]
	var filtered []*domain.ClinicalEvent

	for _, ev := range all {
		if eventTypeFilter == "" || ev.EventType == eventTypeFilter {
			filtered = append(filtered, ev)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].EventTimestamp.After(filtered[j].EventTimestamp)
	})

	if len(filtered) > limit {
		return filtered[:limit], nil
	}
	return filtered, nil
}

func (r *InMemoryClinicalRepo) SaveEvent(ctx context.Context, event *domain.ClinicalEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events[event.PatientRun] = append(r.events[event.PatientRun], event)
	return nil
}

func (r *InMemoryClinicalRepo) RevokeConsent(ctx context.Context, patientRun string, dataCategory string) error {
	return nil
}
