package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SQSClient interface {
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

type snsMessageEnvelope struct {
	Message string `json:"Message"`
}

type ProjectionConsumerConfig struct {
	QueueURL          string
	MaxMessages       int32
	WaitTimeSeconds   int32
	VisibilityTimeout int32
	PollInterval      time.Duration
}

type ProjectionConsumer struct {
	client    SQSClient
	projector wallet.BalanceProjector
	config    ProjectionConsumerConfig
	logger    *slog.Logger
}

func NewProjectionConsumer(
	client SQSClient,
	projector wallet.BalanceProjector,
	cfg ProjectionConsumerConfig,
	logger *slog.Logger,
) (*ProjectionConsumer, error) {
	if client == nil {
		return nil, fmt.Errorf("sqs client cannot be nil")
	}
	if projector == nil {
		return nil, fmt.Errorf("balance projector cannot be nil")
	}
	if cfg.QueueURL == "" {
		return nil, fmt.Errorf("sqs queue url cannot be empty")
	}
	if cfg.MaxMessages <= 0 {
		cfg.MaxMessages = 10
	}
	if cfg.WaitTimeSeconds <= 0 {
		cfg.WaitTimeSeconds = 10
	}
	if cfg.VisibilityTimeout <= 0 {
		cfg.VisibilityTimeout = 30
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 100 * time.Millisecond
	}

	return &ProjectionConsumer{
		client:    client,
		projector: projector,
		config:    cfg,
		logger:    logger,
	}, nil
}

func (c *ProjectionConsumer) ProcessMessages(ctx context.Context) (int, error) {
	output, err := c.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.config.QueueURL),
		MaxNumberOfMessages: c.config.MaxMessages,
		WaitTimeSeconds:     c.config.WaitTimeSeconds,
		VisibilityTimeout:   c.config.VisibilityTimeout,
	})
	if err != nil {
		return 0, fmt.Errorf("receive sqs messages: %w", err)
	}

	processedCount := 0
	for _, msg := range output.Messages {
		if err := c.processMessage(ctx, msg); err != nil {
			if c.logger != nil {
				c.logger.ErrorContext(
					ctx,
					"sqs_message_processing_failed",
					slog.String("message_id", aws.ToString(msg.MessageId)),
					slog.Any("error", err),
				)
			}
			continue
		}
		processedCount++
	}

	return processedCount, nil
}

func (c *ProjectionConsumer) processMessage(ctx context.Context, msg types.Message) error {
	rawBody := aws.ToString(msg.Body)

	var envelope snsMessageEnvelope
	var payloadBytes []byte
	if err := json.Unmarshal([]byte(rawBody), &envelope); err == nil && envelope.Message != "" {
		payloadBytes = []byte(envelope.Message)
	} else {
		payloadBytes = []byte(rawBody)
	}

	var payload wallet.WalletBalanceUpdatedPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	walletID, err := domain.NewWalletID(payload.WalletID)
	if err != nil {
		return fmt.Errorf("invalid wallet ID: %w", err)
	}

	_, err = c.projector.ProjectBalance(ctx, walletID)
	if err != nil {
		return fmt.Errorf("project balance for wallet %q: %w", walletID, err)
	}

	_, err = c.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.config.QueueURL),
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		return fmt.Errorf("delete sqs message %q: %w", aws.ToString(msg.MessageId), err)
	}

	if c.logger != nil {
		c.logger.DebugContext(
			ctx,
			"sqs_wallet_balance_projected",
			slog.String("wallet_id", walletID.String()),
			slog.String("message_id", aws.ToString(msg.MessageId)),
		)
	}

	return nil
}

func (c *ProjectionConsumer) Start(ctx context.Context) error {
	ticker := time.NewTicker(c.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_, _ = c.ProcessMessages(ctx)
		}
	}
}
