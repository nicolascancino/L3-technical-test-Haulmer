package secrets

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type AWSClient struct {
	client *secretsmanager.Client
}

func NewAWSClient(ctx context.Context, region string) (AWSClient, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return AWSClient{}, fmt.Errorf("load aws config: %w", err)
	}
	return AWSClient{client: secretsmanager.NewFromConfig(cfg)}, nil
}

func (c AWSClient) Get(ctx context.Context, secretID string) (string, error) {
	out, err := c.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(secretID)})
	if err != nil {
		return "", fmt.Errorf("get secret value: %w", err)
	}
	if out.SecretString == nil {
		return "", errors.New("secret has no string value")
	}
	return *out.SecretString, nil
}
