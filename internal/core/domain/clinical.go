package domain

import "time"

type PatientSummary struct {
	PatientRun        string
	ActiveAllergies   []string
	ActiveDiagnoses   []string
	ActiveMedications []string
	LastUpdate        time.Time
}

type ClinicalEvent struct {
	EventID          string
	PatientRun       string
	EventType        string
	EventDataJSON    []byte
	AuthorCredential string
	RecordedAt       time.Time
	EventTimestamp   time.Time
}
