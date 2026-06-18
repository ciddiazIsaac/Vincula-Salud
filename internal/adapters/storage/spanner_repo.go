package storage

import (
	"context"
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/minsal/vincula/internal/core/domain"
	"github.com/minsal/vincula/internal/core/ports"
	"go.opentelemetry.io/otel"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SpannerClinicalRepo struct {
	client *spanner.Client
}

func NewSpannerClinicalRepo(ctx context.Context, database string) (ports.ClinicalRecordRepository, *spanner.Client, error) {
	client, err := spanner.NewClient(ctx, database)
	if err != nil {
		return nil, nil, err
	}
	return &SpannerClinicalRepo{client: client}, client, nil
}

func (r *SpannerClinicalRepo) GetRecentEvents(ctx context.Context, patientRun string, limit int) ([]*domain.ClinicalEvent, error) {
	tracer := otel.Tracer("vincula-salud-clinical/adapters")
	ctx, span := tracer.Start(ctx, "GetRecentEvents")
	defer span.End()
	slog.InfoContext(ctx, "Fetching recent events", "patient_run", patientRun, "limit", limit)

	queryCtx, queryCancel := context.WithTimeout(ctx, 5*time.Second)
	defer queryCancel()

	stmt := spanner.Statement{
		SQL: `SELECT event_id, patient_run, event_type, event_data_json, author_credential, recorded_at, event_timestamp 
              FROM clinical_events 
              WHERE patient_run = @run 
              ORDER BY event_timestamp DESC LIMIT @limit`,
		Params: map[string]interface{}{"run": patientRun, "limit": limit},
	}
	iter := r.client.Single().Query(queryCtx, stmt)
	defer iter.Stop()

	var events []*domain.ClinicalEvent
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		ev := &domain.ClinicalEvent{}
		err = row.Columns(&ev.EventID, &ev.PatientRun, &ev.EventType, &ev.EventDataJSON, &ev.AuthorCredential, &ev.RecordedAt, &ev.EventTimestamp)
		if err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, nil
}

func (r *SpannerClinicalRepo) ListEvents(ctx context.Context, patientRun string, eventTypeFilter string, limit int) ([]*domain.ClinicalEvent, error) {
	tracer := otel.Tracer("vincula-salud-clinical/adapters")
	ctx, span := tracer.Start(ctx, "ListEvents")
	defer span.End()
	slog.InfoContext(ctx, "Listing clinical events", "patient_run", patientRun, "event_type", eventTypeFilter)

	sql := "SELECT event_id, patient_run, event_type, event_data_json, author_credential, recorded_at, event_timestamp FROM clinical_events WHERE patient_run = @run"
	params := map[string]interface{}{"run": patientRun}
	if eventTypeFilter != "" {
		sql += " AND event_type = @type"
		params["type"] = eventTypeFilter
	}
	sql += " ORDER BY event_timestamp DESC LIMIT @limit"
	params["limit"] = limit

	listCtx, listCancel := context.WithTimeout(ctx, 5*time.Second)
	defer listCancel()

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := r.client.Single().Query(listCtx, stmt)
	defer iter.Stop()

	var events []*domain.ClinicalEvent
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		ev := &domain.ClinicalEvent{}
		err = row.Columns(&ev.EventID, &ev.PatientRun, &ev.EventType, &ev.EventDataJSON, &ev.AuthorCredential, &ev.RecordedAt, &ev.EventTimestamp)
		if err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, nil
}

func (r *SpannerClinicalRepo) SaveEvent(ctx context.Context, event *domain.ClinicalEvent) error {
	tracer := otel.Tracer("vincula-salud-clinical/adapters")
	ctx, span := tracer.Start(ctx, "SaveEvent")
	defer span.End()

	mutation := spanner.InsertOrUpdate("clinical_events",
		[]string{"event_id", "patient_run", "event_type", "event_data_json", "author_credential", "recorded_at", "event_timestamp"},
		[]interface{}{event.EventID, event.PatientRun, event.EventType, event.EventDataJSON, event.AuthorCredential, event.RecordedAt, event.EventTimestamp})

	applyCtx, applyCancel := context.WithTimeout(ctx, 5*time.Second)
	defer applyCancel()
	_, err := r.client.Apply(applyCtx, []*spanner.Mutation{mutation})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to insert: %v", err)
	}
	return nil
}

func (r *SpannerClinicalRepo) RevokeConsent(ctx context.Context, patientRun string, dataCategory string) error {
	tracer := otel.Tracer("vincula-salud-clinical/adapters")
	ctx, span := tracer.Start(ctx, "RevokeConsent")
	defer span.End()
	slog.InfoContext(ctx, "Revoking consent", "patient_run", patientRun, "category", dataCategory)

	mutation := spanner.InsertOrUpdate("patient_consent",
		[]string{"patient_run", "data_category", "revoked_at"},
		[]interface{}{patientRun, dataCategory, time.Now()})
	applyCtx, applyCancel := context.WithTimeout(ctx, 5*time.Second)
	defer applyCancel()
	_, err := r.client.Apply(applyCtx, []*spanner.Mutation{mutation})
	return err
}
