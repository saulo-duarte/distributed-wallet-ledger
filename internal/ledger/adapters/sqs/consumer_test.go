package sqs

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type mockSQSClient struct {
	messagesToReturn []types.Message
	deletedHandles   []string
	receiveErr       error
	deleteErr        error
}

func (m *mockSQSClient) ReceiveMessage(
	_ context.Context,
	_ *sqs.ReceiveMessageInput,
	_ ...func(*sqs.Options),
) (*sqs.ReceiveMessageOutput, error) {
	if m.receiveErr != nil {
		return nil, m.receiveErr
	}
	return &sqs.ReceiveMessageOutput{
		Messages: m.messagesToReturn,
	}, nil
}

func (m *mockSQSClient) DeleteMessage(
	_ context.Context,
	params *sqs.DeleteMessageInput,
	_ ...func(*sqs.Options),
) (*sqs.DeleteMessageOutput, error) {
	if m.deleteErr != nil {
		return nil, m.deleteErr
	}
	m.deletedHandles = append(m.deletedHandles, aws.ToString(params.ReceiptHandle))
	return &sqs.DeleteMessageOutput{}, nil
}

type mockBalanceProjector struct {
	projectedWalletID domain.WalletID
	calls             int
	err               error
}

func (m *mockBalanceProjector) ProjectBalance(
	_ context.Context,
	walletID domain.WalletID,
) (wallet.WalletBalance, error) {
	m.calls++
	m.projectedWalletID = walletID
	if m.err != nil {
		return wallet.WalletBalance{}, m.err
	}
	return wallet.WalletBalance{}, nil
}

func TestNewProjectionConsumer(t *testing.T) {
	t.Parallel()

	client := &mockSQSClient{}
	projector := &mockBalanceProjector{}

	t.Run("rejects nil client", func(t *testing.T) {
		t.Parallel()
		_, err := NewProjectionConsumer(nil, projector, ProjectionConsumerConfig{QueueURL: "http://sqs"}, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("rejects nil projector", func(t *testing.T) {
		t.Parallel()
		_, err := NewProjectionConsumer(client, nil, ProjectionConsumerConfig{QueueURL: "http://sqs"}, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("rejects empty queue url", func(t *testing.T) {
		t.Parallel()
		_, err := NewProjectionConsumer(client, projector, ProjectionConsumerConfig{}, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestProjectionConsumer_ProcessMessages(t *testing.T) {
	t.Parallel()

	t.Run("processes direct json message successfully", func(t *testing.T) {
		t.Parallel()

		client := &mockSQSClient{
			messagesToReturn: []types.Message{
				{
					MessageId:     aws.String("msg-1"),
					ReceiptHandle: aws.String("receipt-1"),
					Body:          aws.String(`{"wallet_id":"wallet-001"}`),
				},
			},
		}
		projector := &mockBalanceProjector{}
		consumer, err := NewProjectionConsumer(client, projector, ProjectionConsumerConfig{
			QueueURL: "http://sqs/queue",
		}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		count, err := consumer.ProcessMessages(context.Background())
		if err != nil {
			t.Fatalf("unexpected process error: %v", err)
		}

		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
		if projector.calls != 1 {
			t.Errorf("projector calls = %d, want 1", projector.calls)
		}
		if projector.projectedWalletID.String() != "wallet-001" {
			t.Errorf("projected wallet = %q, want wallet-001", projector.projectedWalletID)
		}
		if len(client.deletedHandles) != 1 || client.deletedHandles[0] != "receipt-1" {
			t.Errorf("deleted handles = %v, want ['receipt-1']", client.deletedHandles)
		}
	})

	t.Run("processes SNS-wrapped envelope message successfully", func(t *testing.T) {
		t.Parallel()

		snsWrappedBody := `{"Type":"Notification","Message":"{\"wallet_id\":\"wallet-002\"}"}`
		client := &mockSQSClient{
			messagesToReturn: []types.Message{
				{
					MessageId:     aws.String("msg-2"),
					ReceiptHandle: aws.String("receipt-2"),
					Body:          aws.String(snsWrappedBody),
				},
			},
		}
		projector := &mockBalanceProjector{}
		consumer, err := NewProjectionConsumer(client, projector, ProjectionConsumerConfig{
			QueueURL: "http://sqs/queue",
		}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		count, err := consumer.ProcessMessages(context.Background())
		if err != nil {
			t.Fatalf("unexpected process error: %v", err)
		}

		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
		if projector.projectedWalletID.String() != "wallet-002" {
			t.Errorf("projected wallet = %q, want wallet-002", projector.projectedWalletID)
		}
	})

	t.Run("does not delete message when projection fails", func(t *testing.T) {
		t.Parallel()

		client := &mockSQSClient{
			messagesToReturn: []types.Message{
				{
					MessageId:     aws.String("msg-3"),
					ReceiptHandle: aws.String("receipt-3"),
					Body:          aws.String(`{"wallet_id":"wallet-003"}`),
				},
			},
		}
		projector := &mockBalanceProjector{err: errors.New("dynamo failure")}
		consumer, err := NewProjectionConsumer(client, projector, ProjectionConsumerConfig{
			QueueURL: "http://sqs/queue",
		}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		count, err := consumer.ProcessMessages(context.Background())
		if err != nil {
			t.Fatalf("unexpected process error: %v", err)
		}

		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
		if len(client.deletedHandles) != 0 {
			t.Errorf("expected 0 deleted messages, got %d", len(client.deletedHandles))
		}
	})
}
