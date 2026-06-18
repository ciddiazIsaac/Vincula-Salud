package storage

import (
	"context"
	"time"

	"github.com/minsal/vincula/internal/core/domain"
	"github.com/minsal/vincula/internal/core/ports"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type circuitBreakerRepo struct {
	repo ports.ClinicalRecordRepository
	cb   *gobreaker.CircuitBreaker
}

func NewCircuitBreakerRepo(repo ports.ClinicalRecordRepository) ports.ClinicalRecordRepository {
	st := gobreaker.Settings{
		Name:        "ClinicalRecordRepo",
		MaxRequests: 1,
		Interval:    time.Duration(0),
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	}
	cb := gobreaker.NewCircuitBreaker(st)

	return &circuitBreakerRepo{
		repo: repo,
		cb:   cb,
	}
}

func (c *circuitBreakerRepo) GetRecentEvents(ctx context.Context, patientRun string, limit int) ([]*domain.ClinicalEvent, error) {
	res, err := c.cb.Execute(func() (interface{}, error) {
		return c.repo.GetRecentEvents(ctx, patientRun, limit)
	})
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return nil, status.Error(codes.Unavailable, "service unavailable due to circuit breaker")
		}
		return nil, err
	}
	return res.([]*domain.ClinicalEvent), nil
}

func (c *circuitBreakerRepo) ListEvents(ctx context.Context, patientRun string, eventTypeFilter string, limit int) ([]*domain.ClinicalEvent, error) {
	res, err := c.cb.Execute(func() (interface{}, error) {
		return c.repo.ListEvents(ctx, patientRun, eventTypeFilter, limit)
	})
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return nil, status.Error(codes.Unavailable, "service unavailable due to circuit breaker")
		}
		return nil, err
	}
	return res.([]*domain.ClinicalEvent), nil
}

func (c *circuitBreakerRepo) SaveEvent(ctx context.Context, event *domain.ClinicalEvent) error {
	_, err := c.cb.Execute(func() (interface{}, error) {
		return nil, c.repo.SaveEvent(ctx, event)
	})
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return status.Error(codes.Unavailable, "service unavailable due to circuit breaker")
		}
		return err
	}
	return nil
}

func (c *circuitBreakerRepo) RevokeConsent(ctx context.Context, patientRun string, dataCategory string) error {
	_, err := c.cb.Execute(func() (interface{}, error) {
		return nil, c.repo.RevokeConsent(ctx, patientRun, dataCategory)
	})
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return status.Error(codes.Unavailable, "service unavailable due to circuit breaker")
		}
		return err
	}
	return nil
}
