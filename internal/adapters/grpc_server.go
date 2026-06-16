package adapters

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "sync"
    
    clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
    "google.golang.org/protobuf/types/known/timestamppb"
)

type InMemoryClinicalStore struct {
    clinicalv1.UnimplementedClinicalRecordServiceServer
    mu     sync.RWMutex
    events map[string][]*clinicalv1.ClinicalEvent
}

func NewInMemoryClinicalStore() *InMemoryClinicalStore {
    return &InMemoryClinicalStore{
        events: make(map[string][]*clinicalv1.ClinicalEvent),
    }
}

func (s *InMemoryClinicalStore) GetPatientSummary(ctx context.Context, req *clinicalv1.GetPatientSummaryRequest) (*clinicalv1.PatientSummary, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    events := s.events[req.PatientRun]
    summary := &clinicalv1.PatientSummary{
        PatientRun: req.PatientRun,
        LastUpdate: timestamppb.Now(),
    }
    for _, ev := range events {
        if ev.EventType == "allergy" {
            summary.ActiveAllergies = append(summary.ActiveAllergies, "registrado")
        }
        if ev.EventTimestamp.AsTime().After(summary.LastUpdate.AsTime()) {
            summary.LastUpdate = ev.EventTimestamp
        }
    }
    return summary, nil
}

func (s *InMemoryClinicalStore) RecordClinicalEvent(ctx context.Context, req *clinicalv1.RecordClinicalEventRequest) (*clinicalv1.ClinicalEvent, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    idBytes := make([]byte, 16)
    rand.Read(idBytes)
    event := &clinicalv1.ClinicalEvent{
        EventId:          hex.EncodeToString(idBytes),
        PatientRun:       req.PatientRun,
        EventType:        req.EventType,
        EventDataJson:    req.EventDataJson,
        AuthorCredential: req.AuthorCredential,
        RecordedAt:       timestamppb.Now(),
        EventTimestamp:   req.EventTimestamp,
    }
    if event.EventTimestamp == nil {
        event.EventTimestamp = timestamppb.Now()
    }
    s.events[req.PatientRun] = append(s.events[req.PatientRun], event)
    return event, nil
}

func (s *InMemoryClinicalStore) ListClinicalEvents(context.Context, *clinicalv1.ListClinicalEventsRequest) (*clinicalv1.ListClinicalEventsResponse, error) {
    return nil, nil
}
func (s *InMemoryClinicalStore) RevokeConsent(context.Context, *clinicalv1.RevokeConsentRequest) (*clinicalv1.RevokeConsentResponse, error) {
    return &clinicalv1.RevokeConsentResponse{Success: true}, nil
}
