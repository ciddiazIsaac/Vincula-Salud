package grpc

import (
	"context"
	"time"

	clinicalv1 "github.com/minsal/vincula/api/v1/clinical"
	"github.com/minsal/vincula/internal/core/domain"
	"github.com/minsal/vincula/internal/core/ports"
	"github.com/minsal/vincula/internal/telemetry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ClinicalServer struct {
	clinicalv1.UnimplementedClinicalRecordServiceServer
	useCase ports.ClinicalRecordUseCase
}

func NewClinicalServer(useCase ports.ClinicalRecordUseCase) *ClinicalServer {
	return &ClinicalServer{useCase: useCase}
}

func (s *ClinicalServer) GetPatientSummary(ctx context.Context, req *clinicalv1.GetPatientSummaryRequest) (*clinicalv1.PatientSummary, error) {
	summary, err := s.useCase.GetPatientSummary(ctx, req.PatientRun)
	if err != nil {
		return nil, err
	}

	return &clinicalv1.PatientSummary{
		PatientRun:         summary.PatientRun,
		ActiveAllergies:    summary.ActiveAllergies,
		ActiveDiagnoses:    summary.ActiveDiagnoses,
		ActiveMedications:  summary.ActiveMedications,
		LastUpdate:         timestamppb.New(summary.LastUpdate),
	}, nil
}

func (s *ClinicalServer) RecordClinicalEvent(ctx context.Context, req *clinicalv1.RecordClinicalEventRequest) (*clinicalv1.ClinicalEvent, error) {
	var eventTs time.Time
	if req.EventTimestamp != nil {
		eventTs = req.EventTimestamp.AsTime()
	}

	event := &domain.ClinicalEvent{
		PatientRun:       req.PatientRun,
		EventType:        req.EventType,
		EventDataJSON:    req.EventDataJson,
		AuthorCredential: req.AuthorCredential,
		EventTimestamp:   eventTs,
	}

	savedEvent, err := s.useCase.RecordClinicalEvent(ctx, event)
	if err != nil {
		telemetry.BusinessErrors.Inc()
		return nil, err
	}

	telemetry.EventsRecorded.WithLabelValues(savedEvent.EventType).Inc()

	return &clinicalv1.ClinicalEvent{
		EventId:          savedEvent.EventID,
		PatientRun:       savedEvent.PatientRun,
		EventType:        savedEvent.EventType,
		EventDataJson:    savedEvent.EventDataJSON,
		AuthorCredential: savedEvent.AuthorCredential,
		RecordedAt:       timestamppb.New(savedEvent.RecordedAt),
		EventTimestamp:   timestamppb.New(savedEvent.EventTimestamp),
	}, nil
}

func (s *ClinicalServer) ListClinicalEvents(ctx context.Context, req *clinicalv1.ListClinicalEventsRequest) (*clinicalv1.ListClinicalEventsResponse, error) {
	events, err := s.useCase.ListClinicalEvents(ctx, req.PatientRun, req.EventTypeFilter, int(req.PageSize))
	if err != nil {
		return nil, err
	}

	var pbEvents []*clinicalv1.ClinicalEvent
	for _, ev := range events {
		pbEvents = append(pbEvents, &clinicalv1.ClinicalEvent{
			EventId:          ev.EventID,
			PatientRun:       ev.PatientRun,
			EventType:        ev.EventType,
			EventDataJson:    ev.EventDataJSON,
			AuthorCredential: ev.AuthorCredential,
			RecordedAt:       timestamppb.New(ev.RecordedAt),
			EventTimestamp:   timestamppb.New(ev.EventTimestamp),
		})
	}

	return &clinicalv1.ListClinicalEventsResponse{
		Events: pbEvents,
	}, nil
}

func (s *ClinicalServer) RevokeConsent(ctx context.Context, req *clinicalv1.RevokeConsentRequest) (*clinicalv1.RevokeConsentResponse, error) {
	err := s.useCase.RevokeConsent(ctx, req.PatientRun, req.DataCategory)
	if err != nil {
		return &clinicalv1.RevokeConsentResponse{Success: false, Message: err.Error()}, nil
	}
	return &clinicalv1.RevokeConsentResponse{Success: true}, nil
}
