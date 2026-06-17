package adapters

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
	clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
	"go.opentelemetry.io/otel"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SpannerClinicalStore struct {
	clinicalv1.UnimplementedClinicalRecordServiceServer
	client *spanner.Client
}

func NewSpannerClinicalStore(ctx context.Context, database string) (*SpannerClinicalStore, error) {
	client, err := spanner.NewClient(ctx, database)
	if err != nil {
		return nil, err
	}
	return &SpannerClinicalStore{client: client}, nil
}

func (s *SpannerClinicalStore) Close() {
	s.client.Close()
}

func (s *SpannerClinicalStore) GetPatientSummary(ctx context.Context, req *clinicalv1.GetPatientSummaryRequest) (*clinicalv1.PatientSummary, error) {
	tracer := otel.Tracer("vincula-salud-clinical/adapters")
	ctx, span := tracer.Start(ctx, "GetPatientSummary")
	defer span.End()
	slog.InfoContext(ctx, "Fetching patient summary", "patient_run", req.PatientRun)

	stmt := spanner.Statement{
		SQL: `SELECT event_type, event_data_json FROM clinical_events 
              WHERE patient_run = @run 
              ORDER BY event_timestamp DESC LIMIT 10`,
		Params: map[string]interface{}{"run": req.PatientRun},
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
			return nil, err
		}
		var eventType string
		var dataBytes []byte
		if err := row.Columns(&eventType, &dataBytes); err != nil {
			return nil, err
		}
		var data map[string]interface{}
		json.Unmarshal(dataBytes, &data)
		switch eventType {
		case "allergy":
			if val, ok := data["alergia"]; ok {
				summary.ActiveAllergies = append(summary.ActiveAllergies, val.(string))
			}
		case "diagnosis":
			if val, ok := data["diagnostico"]; ok {
				summary.ActiveDiagnoses = append(summary.ActiveDiagnoses, val.(string))
			}
		case "prescription":
			if val, ok := data["medicamento"]; ok {
				summary.ActiveMedications = append(summary.ActiveMedications, val.(string))
			}
		}
		// update last_update max event_timestamp (simplificado)
	}
	return summary, nil
}

func (s *SpannerClinicalStore) RecordClinicalEvent(ctx context.Context, req *clinicalv1.RecordClinicalEventRequest) (*clinicalv1.ClinicalEvent, error) {
	tracer := otel.Tracer("vincula-salud-clinical/adapters")
	ctx, span := tracer.Start(ctx, "RecordClinicalEvent")
	defer span.End()
	slog.InfoContext(ctx, "Recording clinical event", "patient_run", req.PatientRun, "event_type", req.EventType)

	idBytes := make([]byte, 16)
	rand.Read(idBytes)
	eventID := hex.EncodeToString(idBytes)

	eventTimestamp := req.EventTimestamp
	if eventTimestamp == nil {
		eventTimestamp = timestamppb.Now()
	}
	recordedAt := timestamppb.Now()

	mutation := spanner.InsertOrUpdate("clinical_events",
		[]string{"event_id", "patient_run", "event_type", "event_data_json", "author_credential", "recorded_at", "event_timestamp"},
		[]interface{}{eventID, req.PatientRun, req.EventType, req.EventDataJson, req.AuthorCredential, recordedAt.AsTime(), eventTimestamp.AsTime()})

	_, err := s.client.Apply(ctx, []*spanner.Mutation{mutation})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert: %v", err)
	}

	return &clinicalv1.ClinicalEvent{
		EventId:          eventID,
		PatientRun:       req.PatientRun,
		EventType:        req.EventType,
		EventDataJson:    req.EventDataJson,
		AuthorCredential: req.AuthorCredential,
		RecordedAt:       recordedAt,
		EventTimestamp:   eventTimestamp,
	}, nil
}

func (s *SpannerClinicalStore) ListClinicalEvents(ctx context.Context, req *clinicalv1.ListClinicalEventsRequest) (*clinicalv1.ListClinicalEventsResponse, error) {
	tracer := otel.Tracer("vincula-salud-clinical/adapters")
	ctx, span := tracer.Start(ctx, "ListClinicalEvents")
	defer span.End()
	slog.InfoContext(ctx, "Listing clinical events", "patient_run", req.PatientRun, "event_type", req.EventTypeFilter)

	sql := "SELECT event_id, patient_run, event_type, event_data_json, author_credential, recorded_at, event_timestamp FROM clinical_events WHERE patient_run = @run"
	params := map[string]interface{}{"run": req.PatientRun}
	if req.EventTypeFilter != "" {
		sql += " AND event_type = @type"
		params["type"] = req.EventTypeFilter
	}
	sql += " ORDER BY event_timestamp DESC LIMIT @limit"
	params["limit"] = req.PageSize
	if req.PageSize == 0 {
		params["limit"] = 100
	}
	if req.PageToken != "" {
		// simplified: ignore pagination token for now
	}
	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var events []*clinicalv1.ClinicalEvent
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		ev := &clinicalv1.ClinicalEvent{}
		var recordedAt, eventTimestamp time.Time
		err = row.Columns(&ev.EventId, &ev.PatientRun, &ev.EventType, &ev.EventDataJson, &ev.AuthorCredential, &recordedAt, &eventTimestamp)
		if err != nil {
			return nil, err
		}
		ev.RecordedAt = timestamppb.New(recordedAt)
		ev.EventTimestamp = timestamppb.New(eventTimestamp)
		events = append(events, ev)
	}
	return &clinicalv1.ListClinicalEventsResponse{Events: events}, nil
}

func (s *SpannerClinicalStore) RevokeConsent(ctx context.Context, req *clinicalv1.RevokeConsentRequest) (*clinicalv1.RevokeConsentResponse, error) {
	tracer := otel.Tracer("vincula-salud-clinical/adapters")
	ctx, span := tracer.Start(ctx, "RevokeConsent")
	defer span.End()
	slog.InfoContext(ctx, "Revoking consent", "patient_run", req.PatientRun, "category", req.DataCategory)

	mutation := spanner.InsertOrUpdate("patient_consent",
		[]string{"patient_run", "data_category", "revoked_at"},
		[]interface{}{req.PatientRun, req.DataCategory, time.Now()})
	_, err := s.client.Apply(ctx, []*spanner.Mutation{mutation})
	if err != nil {
		return &clinicalv1.RevokeConsentResponse{Success: false, Message: err.Error()}, nil
	}
	return &clinicalv1.RevokeConsentResponse{Success: true}, nil
}
