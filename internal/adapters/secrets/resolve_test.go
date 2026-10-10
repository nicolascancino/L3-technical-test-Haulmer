package secrets

import (
	"context"
	"errors"
	"testing"
)

type fakeGetter struct {
	value string
	err   error
	asked string
}

func (f *fakeGetter) Get(_ context.Context, secretID string) (string, error) {
	f.asked = secretID
	return f.value, f.err
}

func TestResolve_UsesFallbackWithoutSecretID(t *testing.T) {
	getter := &fakeGetter{value: "from-aws"}

	got, err := Resolve(context.Background(), getter, "", "from-env")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got != "from-env" || getter.asked != "" {
		t.Errorf("got %q, asked %q; want fallback and no AWS call", got, getter.asked)
	}
}

func TestResolve_ReadsSecretWhenIDIsSet(t *testing.T) {
	getter := &fakeGetter{value: "from-aws"}

	got, err := Resolve(context.Background(), getter, "cards/salt", "from-env")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got != "from-aws" || getter.asked != "cards/salt" {
		t.Errorf("got %q, asked %q", got, getter.asked)
	}
}

func TestResolve_ReturnsGetterError(t *testing.T) {
	getter := &fakeGetter{err: errors.New("denied")}

	if _, err := Resolve(context.Background(), getter, "cards/salt", ""); err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}
}
