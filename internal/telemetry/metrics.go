package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EventsRecorded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vincula_clinical_events_recorded_total",
			Help: "Total number of clinical events recorded, partitioned by event type.",
		},
		[]string{"event_type"},
	)

	BusinessErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "vincula_business_errors_total",
			Help: "Total number of business logic or validation errors.",
		},
	)
)
