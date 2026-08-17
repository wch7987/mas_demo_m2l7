package llm_test

import (
	"testing"

	"github.com/cms/mas_demo_m2l7/internal/llm"
)

func TestResolveEndpointRegion(t *testing.T) {
	endpoint, err := llm.ResolveEndpoint("cn", "")
	if err != nil {
		t.Fatal(err)
	}
	if endpoint != "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions" {
		t.Fatalf("unexpected endpoint: %s", endpoint)
	}
}

func TestResolveEndpointBaseURL(t *testing.T) {
	endpoint, err := llm.ResolveEndpoint("cn", "https://example.com/compatible-mode/v1")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://example.com/compatible-mode/v1/chat/completions"
	if endpoint != want {
		t.Fatalf("got %s want %s", endpoint, want)
	}
}

func TestResolveEndpointInvalidRegion(t *testing.T) {
	_, err := llm.ResolveEndpoint("invalid", "")
	if err == nil {
		t.Fatal("expected error for invalid region")
	}
}

func TestNormalizeMultimodalToolResult(t *testing.T) {
	messages := []llm.APIMessage{
		{
			Role:    "tool",
			Content: llm.EncodeStringContent("prefix data:image/png;base64,abc123"),
		},
		{
			Role:    "user",
			Content: llm.EncodeStringContent("describe image"),
		},
	}

	out, flag := llm.NormalizeMultimodalToolResult(messages)
	if !flag {
		t.Fatal("expected multimodal flag")
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(out))
	}
}

func TestValidateMessages(t *testing.T) {
	err := llm.ValidateMessages([]llm.APIMessage{
		{Role: "user", Content: llm.EncodeStringContent("hi")},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = llm.ValidateMessages([]llm.APIMessage{
		{Role: "invalid", Content: llm.EncodeStringContent("hi")},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestContextWindowSize(t *testing.T) {
	if llm.ContextWindowSize("qwen-long") != 200_000 {
		t.Fatal("expected long context window")
	}
	if llm.ContextWindowSize("qwen-plus") != 8192 {
		t.Fatal("expected 8192 context window")
	}
}
