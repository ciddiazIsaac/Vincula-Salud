package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/minsal/vincula/internal/core/domain"
	"github.com/minsal/vincula/internal/core/ports"
)

// ---------------------------------------------------------------------------
// Mock Repository
// ---------------------------------------------------------------------------

type mockRepo struct {
	events       map[string][]*domain.ClinicalEvent
	saveErr      error
	getErr       error
	listErr      error
	revokeErr    error
	saveCalled   bool
	revokeCalled bool
}

func newMockRepo() *mockRepo {
	return &mockRepo{events: make(map[string][]*domain.ClinicalEvent)}
}

func (m *mockRepo) SaveEvent(_ context.Context, event *domain.ClinicalEvent) error {
	m.saveCalled = true
	if m.saveErr != nil {
		return m.saveErr
	}
	m.events[event.PatientRun] = append(m.events[event.PatientRun], event)
	return nil
}

func (m *mockRepo) GetRecentEvents(_ context.Context, patientRun string, limit int) ([]*domain.ClinicalEvent, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	evts := m.events[patientRun]
	if len(evts) > limit {
		return evts[:limit], nil
	}
	return evts, nil
}

func (m *mockRepo) ListEvents(_ context.Context, patientRun string, eventTypeFilter string, limit int) ([]*domain.ClinicalEvent, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	all := m.events[patientRun]
	var filtered []*domain.ClinicalEvent
	for _, ev := range all {
		if eventTypeFilter == "" || ev.EventType == eventTypeFilter {
			filtered = append(filtered, ev)
		}
	}
	if len(filtered) > limit {
		return filtered[:limit], nil
	}
	return filtered, nil
}

func (m *mockRepo) RevokeConsent(_ context.Context, _ string, _ string) error {
	m.revokeCalled = true
	return m.revokeErr
}

// Compile-time check that mockRepo implements the interface.
var _ ports.ClinicalRecordRepository = (*mockRepo)(nil)

// ---------------------------------------------------------------------------
// Mock Publisher
// ---------------------------------------------------------------------------

type mockPublisher struct {
	publishErr    error
	publishCalled bool
	lastEvent     *domain.ClinicalEvent
}

func (m *mockPublisher) PublishClinicalEventRecorded(_ context.Context, event *domain.ClinicalEvent) error {
	m.publishCalled = true
	m.lastEvent = event
	return m.publishErr
}

var _ ports.EventPublisher = (*mockPublisher)(nil)

// ---------------------------------------------------------------------------
// RecordClinicalEvent tests
// ---------------------------------------------------------------------------

func TestRecordClinicalEvent_Success(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	uc := NewClinicalUseCase(repo, pub)

	event := &domain.ClinicalEvent{
		PatientRun:       "12345678-9",
		EventType:        "allergy",
		EventDataJSON:    []byte(`{"alergia":"Penicilina"}`),
		AuthorCredential: "DR-TEST",
		EventTimestamp:   time.Now(),
	}

	result, err := uc.RecordClinicalEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.EventID == "" {
		t.Error("expected non-empty EventID")
	}
	if result.RecordedAt.IsZero() {
		t.Error("expected non-zero RecordedAt")
	}
	if !repo.saveCalled {
		t.Error("expected repo.SaveEvent to be called")
	}
	if !pub.publishCalled {
		t.Error("expected publisher to be called")
	}
}

func TestRecordClinicalEvent_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.saveErr = errors.New("database down")
	uc := NewClinicalUseCase(repo, nil)

	event := &domain.ClinicalEvent{
		PatientRun:    "12345678-9",
		EventType:     "allergy",
		EventDataJSON: []byte(`{}`),
	}

	_, err := uc.RecordClinicalEvent(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
}

func TestRecordClinicalEvent_PublisherError_DoesNotFail(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{publishErr: errors.New("pubsub unavailable")}
	uc := NewClinicalUseCase(repo, pub)

	event := &domain.ClinicalEvent{
		PatientRun:    "12345678-9",
		EventType:     "allergy",
		EventDataJSON: []byte(`{}`),
	}

	result, err := uc.RecordClinicalEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("publisher error should not fail the operation: %v", err)
	}
	if result.EventID == "" {
		t.Error("expected non-empty EventID even when publisher fails")
	}
}

func TestRecordClinicalEvent_NilPublisher(t *testing.T) {
	repo := newMockRepo()
	uc := NewClinicalUseCase(repo, nil)

	event := &domain.ClinicalEvent{
		PatientRun:    "12345678-9",
		EventType:     "allergy",
		EventDataJSON: []byte(`{}`),
	}

	result, err := uc.RecordClinicalEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("nil publisher should not cause error: %v", err)
	}
	if result.EventID == "" {
		t.Error("expected non-empty EventID with nil publisher")
	}
}

func TestRecordClinicalEvent_ZeroTimestamp(t *testing.T) {
	repo := newMockRepo()
	uc := NewClinicalUseCase(repo, nil)

	event := &domain.ClinicalEvent{
		PatientRun:    "12345678-9",
		EventType:     "allergy",
		EventDataJSON: []byte(`{}`),
		// EventTimestamp is zero
	}

	result, err := uc.RecordClinicalEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EventTimestamp.IsZero() {
		t.Error("expected non-zero EventTimestamp when input was zero")
	}
}

// ---------------------------------------------------------------------------
// GetPatientSummary tests
// ---------------------------------------------------------------------------

func TestGetPatientSummary_Allergies(t *testing.T) {
	repo := newMockRepo()
	repo.events["12345678-9"] = []*domain.ClinicalEvent{
		{EventType: "allergy", EventDataJSON: []byte(`{"alergia":"Penicilina"}`), EventTimestamp: time.Now()},
	}
	uc := NewClinicalUseCase(repo, nil)

	summary, err := uc.GetPatientSummary(context.Background(), "12345678-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summary.ActiveAllergies) != 1 {
		t.Fatalf("expected 1 allergy, got %d", len(summary.ActiveAllergies))
	}
	if summary.ActiveAllergies[0] != "Penicilina" {
		t.Errorf("expected allergy 'Penicilina', got '%s'", summary.ActiveAllergies[0])
	}
}

func TestGetPatientSummary_Diagnoses(t *testing.T) {
	repo := newMockRepo()
	repo.events["12345678-9"] = []*domain.ClinicalEvent{
		{EventType: "diagnosis", EventDataJSON: []byte(`{"diagnostico":"Hipertensión"}`), EventTimestamp: time.Now()},
	}
	uc := NewClinicalUseCase(repo, nil)

	summary, err := uc.GetPatientSummary(context.Background(), "12345678-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summary.ActiveDiagnoses) != 1 || summary.ActiveDiagnoses[0] != "Hipertensión" {
		t.Errorf("expected diagnosis 'Hipertensión', got %v", summary.ActiveDiagnoses)
	}
}

func TestGetPatientSummary_Prescriptions(t *testing.T) {
	repo := newMockRepo()
	repo.events["12345678-9"] = []*domain.ClinicalEvent{
		{EventType: "prescription", EventDataJSON: []byte(`{"medicamento":"Losartán 50mg"}`), EventTimestamp: time.Now()},
	}
	uc := NewClinicalUseCase(repo, nil)

	summary, err := uc.GetPatientSummary(context.Background(), "12345678-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summary.ActiveMedications) != 1 || summary.ActiveMedications[0] != "Losartán 50mg" {
		t.Errorf("expected medication 'Losartán 50mg', got %v", summary.ActiveMedications)
	}
}

func TestGetPatientSummary_MalformedJSON(t *testing.T) {
	repo := newMockRepo()
	repo.events["12345678-9"] = []*domain.ClinicalEvent{
		{EventType: "allergy", EventDataJSON: []byte(`not-json`), EventTimestamp: time.Now()},
	}
	uc := NewClinicalUseCase(repo, nil)

	summary, err := uc.GetPatientSummary(context.Background(), "12345678-9")
	if err != nil {
		t.Fatalf("malformed JSON should not cause error: %v", err)
	}
	if len(summary.ActiveAllergies) != 0 {
		t.Error("expected 0 allergies for malformed JSON event")
	}
}

func TestGetPatientSummary_EmptyPatient(t *testing.T) {
	repo := newMockRepo()
	uc := NewClinicalUseCase(repo, nil)

	summary, err := uc.GetPatientSummary(context.Background(), "00000000-0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.PatientRun != "00000000-0" {
		t.Errorf("expected PatientRun '00000000-0', got '%s'", summary.PatientRun)
	}
	if len(summary.ActiveAllergies) != 0 || len(summary.ActiveDiagnoses) != 0 || len(summary.ActiveMedications) != 0 {
		t.Error("expected empty summary for patient with no events")
	}
}

func TestGetPatientSummary_LastUpdate(t *testing.T) {
	older := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	repo := newMockRepo()
	repo.events["12345678-9"] = []*domain.ClinicalEvent{
		{EventType: "allergy", EventDataJSON: []byte(`{"alergia":"A"}`), EventTimestamp: older},
		{EventType: "diagnosis", EventDataJSON: []byte(`{"diagnostico":"B"}`), EventTimestamp: newer},
	}
	uc := NewClinicalUseCase(repo, nil)

	summary, err := uc.GetPatientSummary(context.Background(), "12345678-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !summary.LastUpdate.Equal(newer) {
		t.Errorf("expected LastUpdate %v, got %v", newer, summary.LastUpdate)
	}
}

func TestGetPatientSummary_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.getErr = errors.New("connection refused")
	uc := NewClinicalUseCase(repo, nil)

	_, err := uc.GetPatientSummary(context.Background(), "12345678-9")
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
}

// ---------------------------------------------------------------------------
// ListClinicalEvents tests
// ---------------------------------------------------------------------------

func TestListClinicalEvents_DefaultPageSize(t *testing.T) {
	repo := newMockRepo()
	uc := NewClinicalUseCase(repo, nil)

	// We can't easily inspect the pageSize passed to the repo without more
	// instrumentation, but we verify it doesn't panic and returns no error.
	_, err := uc.ListClinicalEvents(context.Background(), "12345678-9", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListClinicalEvents_WithFilter(t *testing.T) {
	repo := newMockRepo()
	repo.events["12345678-9"] = []*domain.ClinicalEvent{
		{EventType: "allergy", EventDataJSON: []byte(`{}`)},
		{EventType: "diagnosis", EventDataJSON: []byte(`{}`)},
		{EventType: "allergy", EventDataJSON: []byte(`{}`)},
	}
	uc := NewClinicalUseCase(repo, nil)

	events, err := uc.ListClinicalEvents(context.Background(), "12345678-9", "allergy", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 allergy events, got %d", len(events))
	}
}

func TestListClinicalEvents_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.listErr = errors.New("timeout")
	uc := NewClinicalUseCase(repo, nil)

	_, err := uc.ListClinicalEvents(context.Background(), "12345678-9", "", 10)
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
}

// ---------------------------------------------------------------------------
// RevokeConsent tests
// ---------------------------------------------------------------------------

func TestRevokeConsent_Success(t *testing.T) {
	repo := newMockRepo()
	uc := NewClinicalUseCase(repo, nil)

	err := uc.RevokeConsent(context.Background(), "12345678-9", "clinical_history")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.revokeCalled {
		t.Error("expected repo.RevokeConsent to be called")
	}
}

func TestRevokeConsent_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.revokeErr = errors.New("permission denied")
	uc := NewClinicalUseCase(repo, nil)

	err := uc.RevokeConsent(context.Background(), "12345678-9", "clinical_history")
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
}
