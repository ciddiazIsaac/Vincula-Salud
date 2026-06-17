package health

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
)

type AdvancedHealthChecker struct {
	spannerClient *spanner.Client
	lastCheck     time.Time
	status        string
}

func NewAdvancedHealthChecker(client *spanner.Client) *AdvancedHealthChecker {
	return &AdvancedHealthChecker{
		spannerClient: client,
		status:        "initializing",
	}
}

func (h *AdvancedHealthChecker) Check(ctx context.Context) map[string]interface{} {
	h.lastCheck = time.Now()

	// Verificar conexión a Spanner
	spannerOK := true
	if h.spannerClient != nil {
		iter := h.spannerClient.Single().Query(ctx, spanner.Statement{SQL: "SELECT 1"})
		defer iter.Stop()
		_, err := iter.Next()
		if err != nil {
			spannerOK = false
		}
	}

	if spannerOK {
		h.status = "healthy"
	} else {
		h.status = "unhealthy"
	}

	return map[string]interface{}{
		"status":    h.status,
		"spanner":   spannerOK,
		"timestamp": time.Now().Unix(),
		"uptime":    time.Since(h.lastCheck).Seconds(),
	}
}
