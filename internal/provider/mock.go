package provider

import (
	"context"
	"os"
)

// mockProvider returns a canned answer (tests and offline demos):
//
//	WTF_MOCK_RESPONSE="some answer" wtf --provider mock
type mockProvider struct{}

func (mockProvider) Name() string { return "mock" }

func (mockProvider) Complete(ctx context.Context, req Request) (string, error) {
	if m := os.Getenv("WTF_MOCK_RESPONSE"); m != "" {
		return m, nil
	}
	return "mock: the command ran and produced output.", nil
}

func (mockProvider) Models(ctx context.Context) ([]string, error) { return []string{"none"}, nil }
