package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/config"
)

type secretGetter interface {
	Get(ctx context.Context, secretID string) (string, error)
}

// readCloudSecrets fails closed: an unreadable or blank secret is an error.
func readCloudSecrets(ctx context.Context, getter secretGetter, cfg config.Config) (dbPassword, panSalt string, err error) {
	if dbPassword, err = readSecret(ctx, getter, cfg.DBPasswordSecretID); err != nil {
		return "", "", err
	}
	if panSalt, err = readSecret(ctx, getter, cfg.PANSaltSecretID); err != nil {
		return "", "", err
	}
	return dbPassword, panSalt, nil
}

func readSecret(ctx context.Context, getter secretGetter, secretID string) (string, error) {
	value, err := getter.Get(ctx, secretID)
	if err != nil {
		return "", fmt.Errorf("read secret %q: %w", secretID, err)
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("secret %q is empty", secretID)
	}
	return value, nil
}
