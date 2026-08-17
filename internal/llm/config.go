package llm

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultModel      = "qwen3.7-plus"
	defaultImageModel = "qwen3.7-plus"
	defaultTimeout    = 600 * time.Second
	defaultRetryCount = 2
)

// Config holds Aliyun Qwen LLM settings.
type Config struct {
	Model       string
	ImageModel  string
	APIKey      string
	Region      string
	BaseURL     string
	Temperature *float32
	Timeout     time.Duration
	RetryCount  int
	Stop        []string
}

// ConfigFromEnv builds config from environment variables.
func ConfigFromEnv() (*Config, error) {
	apiKey := firstNonEmpty(os.Getenv("QWEN_API_KEY"), os.Getenv("DASHSCOPE_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("API Key 未提供，请设置 QWEN_API_KEY 或 DASHSCOPE_API_KEY")
	}

	cfg := &Config{
		Model:      firstNonEmpty(os.Getenv("QWEN_MODEL"), defaultModel),
		ImageModel: firstNonEmpty(os.Getenv("QWEN_IMAGE_MODEL"), defaultImageModel),
		APIKey:     apiKey,
		Region:     firstNonEmpty(os.Getenv("QWEN_REGION"), "cn"),
		BaseURL:    firstNonEmpty(os.Getenv("DASHSCOPE_BASE_URL"), os.Getenv("QWEN_BASE_URL")),
		Timeout:    defaultTimeout,
		RetryCount: defaultRetryCount,
	}

	if v := os.Getenv("LLM_RETRY_COUNT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("无效的 LLM_RETRY_COUNT: %w", err)
		}
		cfg.RetryCount = n
	}

	if v := os.Getenv("QWEN_TEMPERATURE"); v != "" {
		f, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return nil, fmt.Errorf("无效的 QWEN_TEMPERATURE: %w", err)
		}
		t := float32(f)
		cfg.Temperature = &t
	}

	return cfg, nil
}

// ConfigFromApp builds LLM config from go-zero app config, falling back to env vars.
func ConfigFromApp(model, imageModel, apiKey, region, baseURL string, temperature float64, timeoutSec, retryCount int) (*Config, error) {
	cfg, err := ConfigFromEnv()
	if err != nil {
		cfg = &Config{
			Model:      firstNonEmpty(model, os.Getenv("QWEN_MODEL"), defaultModel),
			ImageModel: firstNonEmpty(imageModel, defaultImageModel),
			Region:     firstNonEmpty(region, "cn"),
			Timeout:    defaultTimeout,
			RetryCount: defaultRetryCount,
		}
	}
	if apiKey != "" {
		cfg.APIKey = apiKey
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API Key 未提供，请在配置或环境变量中设置")
	}
	if model != "" {
		cfg.Model = model
	} else if envModel := os.Getenv("QWEN_MODEL"); envModel != "" {
		cfg.Model = envModel
	}
	if imageModel != "" {
		cfg.ImageModel = imageModel
	}
	if region != "" {
		cfg.Region = region
	}
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}
	if temperature > 0 {
		t := float32(temperature)
		cfg.Temperature = &t
	}
	if timeoutSec > 0 {
		cfg.Timeout = time.Duration(timeoutSec) * time.Second
	}
	if retryCount >= 0 {
		cfg.RetryCount = retryCount
	}
	return cfg, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
