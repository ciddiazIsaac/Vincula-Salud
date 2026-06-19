package domain

import (
	"testing"
	"time"
)

func TestClinicalEvent_ZeroValue(t *testing.T) {
	var ev ClinicalEvent

	if ev.EventID != "" {
		t.Errorf("expected empty EventID, got %q", ev.EventID)
	}
	if ev.PatientRun != "" {
		t.Errorf("expected empty PatientRun, got %q", ev.PatientRun)
	}
	if ev.EventType != "" {
		t.Errorf("expected empty EventType, got %q", ev.EventType)
	}
	if ev.EventDataJSON != nil {
		t.Errorf("expected nil EventDataJSON, got %v", ev.EventDataJSON)
	}
	if ev.AuthorCredential != "" {
		t.Errorf("expected empty AuthorCredential, got %q", ev.AuthorCredential)
	}
	if !ev.RecordedAt.IsZero() {
		t.Errorf("expected zero RecordedAt, got %v", ev.RecordedAt)
	}
	if !ev.EventTimestamp.IsZero() {
		t.Errorf("expected zero EventTimestamp, got %v", ev.EventTimestamp)
	}
}

func TestPatientSummary_ZeroValue(t *testing.T) {
	var ps PatientSummary

	if ps.PatientRun != "" {
		t.Errorf("expected empty PatientRun, got %q", ps.PatientRun)
	}
	if ps.ActiveAllergies != nil {
		t.Errorf("expected nil ActiveAllergies, got %v", ps.ActiveAllergies)
	}
	if ps.ActiveDiagnoses != nil {
		t.Errorf("expected nil ActiveDiagnoses, got %v", ps.ActiveDiagnoses)
	}
	if ps.ActiveMedications != nil {
		t.Errorf("expected nil ActiveMedications, got %v", ps.ActiveMedications)
	}
	if !ps.LastUpdate.IsZero() {
		t.Errorf("expected zero LastUpdate, got %v", ps.LastUpdate)
	}
}

func TestClinicalEvent_Fields(t *testing.T) {
	now := time.Now()
	ev := ClinicalEvent{
		EventID:          "evt-123",
		PatientRun:       "12345678-9",
		EventType:        "allergy",
		EventDataJSON:    []byte(`{"alergia":"Penicilina"}`),
		AuthorCredential: "DR-TEST",
		RecordedAt:       now,
		EventTimestamp:   now,
	}

	if ev.EventID != "evt-123" {
		t.Errorf("EventID = %q, want %q", ev.EventID, "evt-123")
	}
	if ev.PatientRun != "12345678-9" {
		t.Errorf("PatientRun = %q, want %q", ev.PatientRun, "12345678-9")
	}
	if ev.EventType != "allergy" {
		t.Errorf("EventType = %q, want %q", ev.EventType, "allergy")
	}
	if string(ev.EventDataJSON) != `{"alergia":"Penicilina"}` {
		t.Errorf("EventDataJSON = %q, want %q", string(ev.EventDataJSON), `{"alergia":"Penicilina"}`)
	}
	if ev.AuthorCredential != "DR-TEST" {
		t.Errorf("AuthorCredential = %q, want %q", ev.AuthorCredential, "DR-TEST")
	}
	if !ev.RecordedAt.Equal(now) {
		t.Errorf("RecordedAt = %v, want %v", ev.RecordedAt, now)
	}
	if !ev.EventTimestamp.Equal(now) {
		t.Errorf("EventTimestamp = %v, want %v", ev.EventTimestamp, now)
	}
}

func TestPatientSummary_Fields(t *testing.T) {
	now := time.Now()
	ps := PatientSummary{
		PatientRun:        "12345678-9",
		ActiveAllergies:   []string{"Penicilina", "Ibuprofeno"},
		ActiveDiagnoses:   []string{"Hipertensión"},
		ActiveMedications: []string{"Losartán 50mg"},
		LastUpdate:        now,
	}

	if ps.PatientRun != "12345678-9" {
		t.Errorf("PatientRun = %q, want %q", ps.PatientRun, "12345678-9")
	}
	if len(ps.ActiveAllergies) != 2 {
		t.Errorf("expected 2 allergies, got %d", len(ps.ActiveAllergies))
	}
	if len(ps.ActiveDiagnoses) != 1 {
		t.Errorf("expected 1 diagnosis, got %d", len(ps.ActiveDiagnoses))
	}
	if len(ps.ActiveMedications) != 1 {
		t.Errorf("expected 1 medication, got %d", len(ps.ActiveMedications))
	}
	if !ps.LastUpdate.Equal(now) {
		t.Errorf("LastUpdate = %v, want %v", ps.LastUpdate, now)
	}
}
