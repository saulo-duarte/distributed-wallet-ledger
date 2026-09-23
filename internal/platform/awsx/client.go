package awsx

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type Config struct {
	Endpoint string
	Region   string
}

func LoadAWSConfig(ctx context.Context, cfg Config) (aws.Config, error) {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		),
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load aws default config: %w", err)
	}

	return awsCfg, nil
}

func NewSNSClient(awsCfg aws.Config, endpoint string) *sns.Client {
	return sns.NewFromConfig(awsCfg, func(o *sns.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})
}

func NewSQSClient(awsCfg aws.Config, endpoint string) *sqs.Client {
	return sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})
}

func NewSSMClient(awsCfg aws.Config, endpoint string) *ssm.Client {
	return ssm.NewFromConfig(awsCfg, func(o *ssm.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})
}
