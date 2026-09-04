package llm

import "testing"

func TestNewClient(t *testing.T) {
	client := NewClient("test-token")
	if client.token != "test-token" {
		t.Fatalf("expected token test-token, got %q", client.token)
	}
	if client.model != "openai/gpt-oss-120b" {
		t.Fatalf("expected model openai/gpt-oss-120b, got %q", client.model)
	}
	if client.endpoint != "https://api.groq.com/openai/v1/chat/completions" {
		t.Fatalf("expected endpoint https://api.groq.com/openai/v1/chat/completions, got %q", client.endpoint)
	}
}
