package adapters

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/timestamppb"

	clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
)

type SpannerStore struct {
	clinicalv1.UnimplementedClinicalRecordServiceServer
	client *spanner.Client
}

func NewSpannerStore(ctx context.Context, dbString string) (*SpannerStore, error) {
	client, err := spanner.NewClient(ctx, dbString)
	if err != nil {
		return nil, fmt.Errorf("failed to create spanner client: %w", err)
	}
	return &SpannerStore{client: client}, nil
}

func (s *SpannerStore) Close() {
	s.client.Close()
}

func (s *SpannerStore) GetPatientSummary(ctx context.Context, req *clinicalv1.GetPatientSummaryRequest) (*clinicalv1.PatientSummary, error) {
	stmt := spanner.Statement{
		SQL: `SELECT EventId, EventType, EventTimestamp 
              FROM ClinicalEvents 
              WHERE PatientRun = @patientRun`,
		Params: map[string]interface{}{
			"patientRun": req.PatientRun,
		},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	summary := &clinicalv1.PatientSummary{
		PatientRun: req.PatientRun,
		LastUpdate: timestamppb.Now(),
	}

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query error: %w", err)
		}

		var eventId, eventType string
		var eventTs spanner.NullTime

		if err := row.Columns(&eventId, &eventType, &eventTs); err != nil {
			return nil, fmt.Errorf("failed to parse row: %w", err)
		}

		if eventType == "allergy" {
			summary.ActiveAllergies = append(summary.ActiveAllergies, "registrado")
		}

		if eventTs.Valid && eventTs.Time.After(summary.LastUpdate.AsTime()) {
			summary.LastUpdate = timestamppb.New(eventTs.Time)
		}
	}

	return summary, nil
}

func (s *SpannerStore) RecordClinicalEvent(ctx context.Context, req *clinicalv1.RecordClinicalEventRequest) (*clinicalv1.ClinicalEvent, error) {
	idBytes := make([]byte, 16)
	rand.Read(idBytes)
	eventId := hex.EncodeToString(idBytes)

	recordedAt := timestamppb.Now()
	eventTs := req.EventTimestamp
	if eventTs == nil {
		eventTs = timestamppb.Now()
	}

	event := &clinicalv1.ClinicalEvent{
		EventId:          eventId,
		PatientRun:       req.PatientRun,
		EventType:        req.EventType,
		EventDataJson:    req.EventDataJson,
		AuthorCredential: req.AuthorCredential,
		RecordedAt:       recordedAt,
		EventTimestamp:   eventTs,
	}

	m := spanner.Insert("ClinicalEvents",
		[]string{"PatientRun", "EventId", "EventType", "EventDataJson", "AuthorCredential", "RecordedAt", "EventTimestamp"},
		[]interface{}{
			event.PatientRun,
			event.EventId,
			event.EventType,
			event.EventDataJson,
			event.AuthorCredential,
			event.RecordedAt.AsTime(),
			event.EventTimestamp.AsTime(),
		})

	_, err := s.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return nil, fmt.Errorf("failed to insert clinical event: %w", err)
	}

	return event, nil
}

func (s *SpannerStore) ListClinicalEvents(ctx context.Context, req *clinicalv1.ListClinicalEventsRequest) (*clinicalv1.ListClinicalEventsResponse, error) {
	return nil, nil
}

func (s *SpannerStore) RevokeConsent(ctx context.Context, req *clinicalv1.RevokeConsentRequest) (*clinicalv1.RevokeConsentResponse, error) {
	return &clinicalv1.RevokeConsentResponse{Success: true}, nil
}
