package llm

import "encoding/json"

// APIMessage is an OpenAI-compatible chat message.
type APIMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content,omitempty"`
	Name       string          `json:"name,omitempty"`
	ToolCalls  []APIToolCall   `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

// APITool describes a function tool for the API.
type APITool struct {
	Type     string           `json:"type"`
	Function APIFunctionDef   `json:"function"`
}

// APIFunctionDef is the function schema in a tool definition.
type APIFunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// APIToolCall is a tool call returned by the assistant.
type APIToolCall struct {
	ID       string            `json:"id"`
	Type     string            `json:"type,omitempty"`
	Function APIFunctionCall   `json:"function"`
}

// APIFunctionCall holds the function name and serialized arguments.
type APIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatCompletionRequest is the request body for /chat/completions.
type ChatCompletionRequest struct {
	Model       string       `json:"model"`
	Messages    []APIMessage `json:"messages"`
	Temperature *float32     `json:"temperature,omitempty"`
	Stop        []string     `json:"stop,omitempty"`
	Tools       []APITool    `json:"tools,omitempty"`
	Stream      bool         `json:"stream,omitempty"`
}

// ChatCompletionResponse is the response body from /chat/completions.
type ChatCompletionResponse struct {
	Choices []struct {
		Message APIAssistantMessage `json:"message"`
	} `json:"choices"`
}

// APIAssistantMessage is the assistant message in a completion response.
type APIAssistantMessage struct {
	Role      string        `json:"role"`
	Content   string        `json:"content"`
	ToolCalls []APIToolCall `json:"tool_calls,omitempty"`
}

// MessageContentPart is one part of multimodal message content.
type MessageContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image_url,omitempty"`
	Image string `json:"image,omitempty"`
}

func EncodeStringContent(s string) json.RawMessage {
	raw, _ := json.Marshal(s)
	return raw
}

func encodeStringContent(s string) json.RawMessage {
	return EncodeStringContent(s)
}

func encodeContentParts(parts []MessageContentPart) json.RawMessage {
	raw, _ := json.Marshal(parts)
	return raw
}

func decodeContentString(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return "", false
		}
		return s, true
	}
	return "", false
}
