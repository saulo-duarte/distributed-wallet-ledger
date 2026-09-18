package awsx

import (
	"context"
	"testing"
)

func TestLoadAWSConfig(t *testing.T) {
	t.Parallel()

	cfg, err := LoadAWSConfig(context.Background(), Config{
		Endpoint: "http://localhost:4566",
		Region:   "us-east-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Region != "us-east-1" {
		t.Errorf("region = %q, want us-east-1", cfg.Region)
	}

	snsClient := NewSNSClient(cfg, "http://localhost:4566")
	if snsClient == nil {
		t.Fatal("expected non-nil sns client")
	}

	sqsClient := NewSQSClient(cfg, "http://localhost:4566")
	if sqsClient == nil {
		t.Fatal("expected non-nil sqs client")
	}
}
