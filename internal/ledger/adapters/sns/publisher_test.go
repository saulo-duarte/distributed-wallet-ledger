package sns

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/domain"

	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type mockSNSClient struct {
	lastInput *sns.PublishInput
	err       error
	calls     int
}

func (m *mockSNSClient) Publish(
	_ context.Context,
	params *sns.PublishInput,
	_ ...func(*sns.Options),
) (*sns.PublishOutput, error) {
	m.calls++
	m.lastInput = params
	if m.err != nil {
		return nil, m.err
	}
	return &sns.PublishOutput{}, nil
}

func TestNewEventPublisher(t *testing.T) {
	t.Parallel()

	client := &mockSNSClient{}

	t.Run("rejects empty topic arn", func(t *testing.T) {
		t.Parallel()
		_, err := NewEventPublisher(client, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("rejects nil client", func(t *testing.T) {
		t.Parallel()
		_, err := NewEventPublisher(nil, "arn:aws:sns:topic")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("creates publisher successfully", func(t *testing.T) {
		t.Parallel()
		pub, err := NewEventPublisher(client, "arn:aws:sns:topic")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pub == nil {
			t.Fatal("expected publisher, got nil")
		}
	})
}

func TestEventPublisher_Publish(t *testing.T) {
	t.Parallel()

	eventID, err := domain.NewEventID("evt-001")
	if err != nil {
		t.Fatalf("unexpected event id error: %v", err)
	}

	event, err := domain.NewOutboxEvent(
		eventID,
		"wallet",
		"w-001",
		"WalletBalanceUpdated",
		[]byte(`{"wallet_id":"w-001"}`),
	)
	if err != nil {
		t.Fatalf("unexpected event creation error: %v", err)
	}

	t.Run("publishes message with attributes to sns", func(t *testing.T) {
		t.Parallel()

		client := &mockSNSClient{}
		pub, err := NewEventPublisher(client, "arn:aws:sns:us-east-1:000:ledger-events")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = pub.Publish(context.Background(), event)
		if err != nil {
			t.Fatalf("unexpected publish error: %v", err)
		}

		if client.calls != 1 {
			t.Errorf("calls = %d, want 1", client.calls)
		}
		if *client.lastInput.TopicArn != "arn:aws:sns:us-east-1:000:ledger-events" {
			t.Errorf("topic = %q, want arn", *client.lastInput.TopicArn)
		}
		if *client.lastInput.Message != string(event.Payload()) {
			t.Errorf("message = %q, want %q", *client.lastInput.Message, string(event.Payload()))
		}
		if *client.lastInput.MessageAttributes["event_type"].StringValue != "WalletBalanceUpdated" {
			t.Errorf("event_type attribute = %q", *client.lastInput.MessageAttributes["event_type"].StringValue)
		}
	})

	t.Run("propagates sns client error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("sns unavailable")
		client := &mockSNSClient{err: expectedErr}
		pub, err := NewEventPublisher(client, "arn:aws:sns:us-east-1:000:ledger-events")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = pub.Publish(context.Background(), event)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("error = %v, want %v", err, expectedErr)
		}
	})
}
