package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// Client calls Aliyun DashScope OpenAI-compatible chat completions API.
type Client struct {
	endpoint   string
	apiKey     string
	model      string
	imageModel string
	temperature *float32
	stop       []string
	timeout    time.Duration
	retryCount int
	httpClient *http.Client
}

// NewClient creates an HTTP client from config.
func NewClient(cfg *Config) (*Client, error) {
	endpoint, err := ResolveEndpoint(cfg.Region, cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	imageModel := cfg.ImageModel
	if imageModel == "" {
		imageModel = defaultImageModel
	}
	retryCount := cfg.RetryCount
	if retryCount < 0 {
		retryCount = defaultRetryCount
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		endpoint:    endpoint,
		apiKey:      cfg.APIKey,
		model:       cfg.Model,
		imageModel:  imageModel,
		temperature: cfg.Temperature,
		stop:        cfg.Stop,
		timeout:     timeout,
		retryCount:  retryCount,
		httpClient:  &http.Client{Timeout: timeout},
	}, nil
}

// ChatCompletion sends a chat completion request with retry and multimodal normalization.
// modelOverride, when non-empty, takes precedence over the configured default model.
func (c *Client) ChatCompletion(ctx context.Context, messages []APIMessage, tools []APITool, modelOverride string, stop []string) (APIAssistantMessage, error) {
	if len(messages) == 0 {
		return APIAssistantMessage{}, fmt.Errorf("messages required")
	}

	body, modelName, err := c.buildChatRequest(messages, tools, modelOverride, stop)
	if err != nil {
		return APIAssistantMessage{}, err
	}

	logx.Infof("发送 LLM API 请求 endpoint=%s model=%s", c.endpoint, modelName)

	var result ChatCompletionResponse
	if err := c.postWithRetry(ctx, body, &result); err != nil {
		return APIAssistantMessage{}, err
	}

	if len(result.Choices) == 0 {
		return APIAssistantMessage{}, fmt.Errorf("响应中未找到 choices 字段")
	}

	assistant := result.Choices[0].Message
	if assistant.Content == "" && len(assistant.ToolCalls) == 0 {
		return APIAssistantMessage{}, fmt.Errorf("LLM 返回空内容，可能是模型限流或偶发异常")
	}

	logx.Infof("收到 LLM API 响应 content_len=%d tool_calls=%d", len(assistant.Content), len(assistant.ToolCalls))
	return assistant, nil
}

func (c *Client) postWithRetry(ctx context.Context, body ChatCompletionRequest, dest *ChatCompletionResponse) error {
	var lastErr error
	maxAttempts := c.retryCount + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := c.postOnce(ctx, body, dest)
		if err == nil {
			if attempt > 0 {
				logx.Infof("llm_request_success_after_retry attempt=%d total=%d", attempt+1, maxAttempts)
			}
			return nil
		}

		lastErr = err
		if attempt >= c.retryCount {
			break
		}

		if retryable(err) {
			logx.Infof("llm_request_retry error=%v attempt=%d max=%d", err, attempt+1, maxAttempts)
			continue
		}
		break
	}

	return lastErr
}

func (c *Client) postOnce(ctx context.Context, body ChatCompletionRequest, dest *ChatCompletionResponse) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &retryableError{msg: fmt.Sprintf("LLM 请求失败: %v", err)}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	switch {
	case resp.StatusCode >= 500:
		return &retryableError{msg: fmt.Sprintf("LLM 服务器错误 %d: %s", resp.StatusCode, truncate(raw, 200))}
	case resp.StatusCode == 429:
		return &retryableError{msg: fmt.Sprintf("LLM 请求限流: %s", truncate(raw, 200))}
	case resp.StatusCode >= 400:
		logx.Errorf("llm_request_4xx status_code=%d body=%s", resp.StatusCode, truncate(raw, 500))
		if resp.StatusCode == 403 {
			return fmt.Errorf("LLM 模型访问被拒绝(403): %s；请检查 Model 配置或请求体 model 字段，确认账号已开通该模型", truncate(raw, 500))
		}
		return fmt.Errorf("LLM 请求错误 %d: %s", resp.StatusCode, truncate(raw, 500))
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

type retryableError struct {
	msg string
}

func (e *retryableError) Error() string { return e.msg }

func retryable(err error) bool {
	_, ok := err.(*retryableError)
	return ok
}

func truncate(raw []byte, n int) string {
	text := strings.TrimSpace(string(raw))
	if len(text) > n {
		return text[:n] + "..."
	}
	return text
}

// ContextWindowSize returns an estimated context window for the model name.
func ContextWindowSize(model string) int {
	m := strings.ToLower(model)
	if strings.Contains(m, "long") {
		return 200_000
	}
	if strings.Contains(m, "max") || strings.Contains(m, "plus") || strings.Contains(m, "turbo") || strings.Contains(m, "flash") {
		return 8192
	}
	return 8192
}

// Model returns the configured text model name.
func (c *Client) Model() string { return c.model }

// ImageModel returns the configured vision model name.
func (c *Client) ImageModel() string { return c.imageModel }
