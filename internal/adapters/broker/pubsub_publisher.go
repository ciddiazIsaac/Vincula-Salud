package broker

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/minsal/vincula/internal/core/domain"
)

type PubSubPublisher struct {
	client *pubsub.Client
	topic  *pubsub.Topic
}

// NewPubSubPublisher creates a new PubSubPublisher connected to the given project and topic.
func NewPubSubPublisher(projectID, topicID string) (*PubSubPublisher, error) {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("pubsub.NewClient: %w", err)
	}

	topic := client.Topic(topicID)
	return &PubSubPublisher{
		client: client,
		topic:  topic,
	}, nil
}

// PublishClinicalEventRecorded publishes the ClinicalEvent to the Pub/Sub topic asynchronously.
func (p *PubSubPublisher) PublishClinicalEventRecorded(ctx context.Context, event *domain.ClinicalEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	msg := &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"patient_run": event.PatientRun,
			"event_type":  event.EventType,
		},
	}

	result := p.topic.Publish(ctx, msg)
	// We wait for the publish to complete to guarantee delivery.
	// In a fully async system, we could just fire and forget, but for clinical data,
	// strict delivery is often preferred.
	_, err = result.Get(ctx)
	if err != nil {
		return fmt.Errorf("topic.Publish.Get: %w", err)
	}

	return nil
}

func (p *PubSubPublisher) Close() error {
	p.topic.Stop()
	return p.client.Close()
}
