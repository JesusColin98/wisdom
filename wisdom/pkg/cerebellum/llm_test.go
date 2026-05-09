package cerebellum

import (
	"context"
	"strings"
	"testing"
)

func TestMockLLM_Complete(t *testing.T) {
	ctx := context.Background()

	t.Run("default response", func(t *testing.T) {
		mock := &MockLLM{}
		resp, err := mock.Complete(ctx, "hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp != "I am a mock LLM response." {
			t.Errorf("expected default response, got %q", resp)
		}
	})

	t.Run("canned response", func(t *testing.T) {
		expected := "canned"
		mock := &MockLLM{CannedResponse: expected}
		resp, err := mock.Complete(ctx, "hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp != expected {
			t.Errorf("expected %q, got %q", expected, resp)
		}
	})

	t.Run("echo", func(t *testing.T) {
		prompt := "hello world"
		mock := &MockLLM{Echo: true}
		resp, err := mock.Complete(ctx, prompt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(resp, prompt) {
			t.Errorf("expected response to contain %q, got %q", prompt, resp)
		}
	})

	t.Run("fail next", func(t *testing.T) {
		mock := &MockLLM{FailNext: true}
		_, err := mock.Complete(ctx, "hello")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		
		// Second call should succeed
		resp, err := mock.Complete(ctx, "hello")
		if err != nil {
			t.Fatalf("unexpected error on second call: %v", err)
		}
		if resp == "" {
			t.Error("expected non-empty response")
		}
	})
}
