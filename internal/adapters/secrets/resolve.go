package secrets

import "context"

type Getter interface {
	Get(ctx context.Context, secretID string) (string, error)
}

// Resolve reads the secret when secretID is set and otherwise returns fallback.
func Resolve(ctx context.Context, getter Getter, secretID, fallback string) (string, error) {
	if secretID == "" {
		return fallback, nil
	}
	return getter.Get(ctx, secretID)
}
