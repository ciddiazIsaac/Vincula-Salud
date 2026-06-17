package health

import (
	"context"
	"sync/atomic"
)

type HealthStatus int32

const (
	StatusUnknown HealthStatus = iota
	StatusServing
	StatusNotServing
)

type HealthChecker struct {
	status atomic.Int32
}

func NewHealthChecker() *HealthChecker {
	hc := &HealthChecker{}
	hc.status.Store(int32(StatusServing))
	return hc
}

func (h *HealthChecker) SetStatus(status HealthStatus) {
	h.status.Store(int32(status))
}

func (h *HealthChecker) IsReady(ctx context.Context) bool {
	return h.status.Load() == int32(StatusServing)
}

func (h *HealthChecker) IsLive(ctx context.Context) bool {
	// Siempre true si el proceso está corriendo
	return true
}
