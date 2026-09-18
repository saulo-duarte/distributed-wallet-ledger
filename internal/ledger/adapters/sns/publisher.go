package sns

import (
	"context"
	"fmt"

	"financial-ledger/internal/ledger/application/outbox"
	"financial-ledger/internal/ledger/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type SNSClient interface {
	Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

type EventPublisher struct {
	client   SNSClient
	topicARN string
}

func NewEventPublisher(client SNSClient, topicARN string) (*EventPublisher, error) {
	if topicARN == "" {
		return nil, fmt.Errorf("sns topic arn cannot be empty")
	}
	if client == nil {
		return nil, fmt.Errorf("sns client cannot be nil")
	}
	return &EventPublisher{
		client:   client,
		topicARN: topicARN,
	}, nil
}

var _ outbox.EventPublisher = (*EventPublisher)(nil)

func (p *EventPublisher) Publish(ctx context.Context, event domain.OutboxEvent) error {
	input := &sns.PublishInput{
		TopicArn: aws.String(p.topicARN),
		Message:  aws.String(string(event.Payload())),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"event_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String(event.ID().String()),
			},
			"event_type": {
				DataType:    aws.String("String"),
				StringValue: aws.String(event.EventType()),
			},
			"aggregate_type": {
				DataType:    aws.String("String"),
				StringValue: aws.String(event.AggregateType()),
			},
			"aggregate_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String(event.AggregateID()),
			},
		},
	}

	_, err := p.client.Publish(ctx, input)
	if err != nil {
		return fmt.Errorf("publish event %q to sns: %w", event.ID(), err)
	}

	return nil
}
