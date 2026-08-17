package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

// StreamDeltaHandler is called for each streamed text delta.
type StreamDeltaHandler func(delta string) error

// chatStreamChunk is one SSE chunk from OpenAI-compatible streaming API.
type chatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// ChatCompletionStream calls /chat/completions with stream=true and invokes onDelta per token chunk.
func (c *Client) ChatCompletionStream(
	ctx context.Context,
	messages []APIMessage,
	tools []APITool,
	modelOverride string,
	onDelta StreamDeltaHandler,
) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("messages required")
	}
	if onDelta == nil {
		return "", fmt.Errorf("onDelta handler required")
	}

	body, modelName, err := c.buildChatRequest(messages, tools, modelOverride, nil)
	if err != nil {
		return "", err
	}
	body.Stream = true

	logx.Infof("发送 LLM 流式 API 请求 endpoint=%s model=%s", c.endpoint, modelName)

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("LLM 流式请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == 403 {
			return "", fmt.Errorf("LLM 模型访问被拒绝(403): %s", truncate(raw, 500))
		}
		return "", fmt.Errorf("LLM 流式请求错误 %d: %s", resp.StatusCode, truncate(raw, 500))
	}

	if err := readSSEStream(resp.Body, onDelta); err != nil {
		return "", err
	}

	logx.Infof("LLM 流式 API 响应完成 model=%s", modelName)
	return modelName, nil
}

func (c *Client) buildChatRequest(messages []APIMessage, tools []APITool, modelOverride string, stop []string) (ChatCompletionRequest, string, error) {
	normalized, useImageModel := NormalizeMultimodalToolResult(messages)
	if err := ValidateMessages(normalized); err != nil {
		return ChatCompletionRequest{}, "", err
	}

	modelName := c.model
	if modelOverride != "" {
		modelName = modelOverride
	}
	if useImageModel {
		modelName = c.imageModel
		if modelOverride != "" {
			modelName = modelOverride
		}
	}

	body := ChatCompletionRequest{
		Model:    modelName,
		Messages: normalized,
	}
	if c.temperature != nil {
		body.Temperature = c.temperature
	}
	if len(c.stop) > 0 && len(stop) == 0 {
		body.Stop = c.stop
	}
	if len(stop) > 0 {
		body.Stop = stop
	}
	if len(tools) > 0 {
		body.Tools = tools
	}
	return body, modelName, nil
}

func readSSEStream(body io.Reader, onDelta StreamDeltaHandler) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}

		var chunk chatStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("decode stream chunk: %w", err)
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		if err := onDelta(delta); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stream: %w", err)
	}
	return nil
}
