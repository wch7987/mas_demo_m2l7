package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// AliyunLLM wraps the HTTP client and implements function-calling loops.
type AliyunLLM struct {
	client *Client
	tools  []*schema.ToolInfo
}

// NewAliyunLLM creates an Aliyun Qwen LLM from config.
func NewAliyunLLM(cfg *Config) (*AliyunLLM, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &AliyunLLM{client: client}, nil
}

// Call sends messages to the LLM and returns text or tool calls.
// modelOverride, when non-empty, overrides the configured default model for this request.
func (llm *AliyunLLM) Call(
	ctx context.Context,
	messages []APIMessage,
	tools []APITool,
	availableFunctions map[string]func(args map[string]any) (string, error),
	maxIterations int,
	modelOverride string,
) (any, error) {
	if maxIterations <= 0 {
		return nil, fmt.Errorf("function calling 达到最大迭代次数，可能存在无限循环")
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages required")
	}

	apiTools := tools
	if len(apiTools) == 0 && len(llm.tools) > 0 {
		converted, err := toolInfosToAPI(llm.tools)
		if err != nil {
			return nil, err
		}
		apiTools = converted
	}

	assistant, err := llm.client.ChatCompletion(ctx, messages, apiTools, modelOverride, nil)
	if err != nil {
		return nil, err
	}

	if len(assistant.ToolCalls) > 0 {
		if availableFunctions != nil {
			return llm.handleFunctionCalls(ctx, assistant.ToolCalls, messages, apiTools, availableFunctions, maxIterations-1, modelOverride)
		}
		return assistant.ToolCalls, nil
	}

	if assistant.Content == "" {
		return nil, fmt.Errorf("LLM 返回空内容")
	}
	return assistant.Content, nil
}

func (llm *AliyunLLM) handleFunctionCalls(
	ctx context.Context,
	toolCalls []APIToolCall,
	messages []APIMessage,
	tools []APITool,
	availableFunctions map[string]func(args map[string]any) (string, error),
	maxIterations int,
	modelOverride string,
) (any, error) {
	messages = append(messages, APIMessage{
		Role:      "assistant",
		ToolCalls: toolCalls,
	})

	for _, toolCall := range toolCalls {
		fnName := toolCall.Function.Name
		toolCallID := toolCall.ID
		if toolCallID == "" {
			return nil, fmt.Errorf("tool_call 缺少 id: %+v", toolCall)
		}

		fn, ok := availableFunctions[fnName]
		if !ok {
			messages = append(messages, APIMessage{
				Role:       "tool",
				ToolCallID: toolCallID,
				Content:    encodeStringContent(fmt.Sprintf("函数 %s 不可用", fnName)),
			})
			continue
		}

		args := map[string]any{}
		raw := toolCall.Function.Arguments
		if raw != "" {
			if err := json.Unmarshal([]byte(raw), &args); err != nil {
				return nil, fmt.Errorf("无法解析函数参数: %w", err)
			}
		}

		result, err := fn(args)
		if err != nil {
			result = "函数执行错误: " + err.Error()
		}
		messages = append(messages, APIMessage{
			Role:       "tool",
			ToolCallID: toolCallID,
			Content:    encodeStringContent(result),
		})
	}

	return llm.Call(ctx, messages, tools, availableFunctions, maxIterations, modelOverride)
}

// Client returns the underlying HTTP client.
func (llm *AliyunLLM) Client() *Client { return llm.client }

// einoChatModel adapts AliyunLLM to eino's ChatModel interface.
type einoChatModel struct {
	llm   *AliyunLLM
	tools []*schema.ToolInfo
}

// NewEinoChatModel creates an eino ChatModel backed by AliyunLLM.
func NewEinoChatModel(cfg *Config) (model.ChatModel, error) {
	llm, err := NewAliyunLLM(cfg)
	if err != nil {
		return nil, err
	}
	return &einoChatModel{llm: llm}, nil
}

func (m *einoChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	apiMessages, err := schemaMessagesToAPI(input)
	if err != nil {
		return nil, err
	}

	options := model.GetCommonOptions(&model.Options{}, opts...)
	toolInfos := m.tools
	if options.Tools != nil {
		toolInfos = options.Tools
	}
	apiTools, err := toolInfosToAPI(toolInfos)
	if err != nil {
		return nil, err
	}

	modelOverride := ""
	if options.Model != nil {
		modelOverride = *options.Model
	}

	assistant, err := m.llm.client.ChatCompletion(ctx, apiMessages, apiTools, modelOverride, options.Stop)
	if err != nil {
		return nil, err
	}
	return apiAssistantToSchema(assistant), nil
}

// WithModel wraps a ChatModel so every Generate/Stream call uses the given model name.
func WithModel(cm model.ChatModel, name string) model.ChatModel {
	if cm == nil || name == "" {
		return cm
	}
	return &chatModelWithModel{ChatModel: cm, name: name}
}

type chatModelWithModel struct {
	model.ChatModel
	name string
}

func (m *chatModelWithModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return m.ChatModel.Generate(ctx, input, append([]model.Option{model.WithModel(m.name)}, opts...)...)
}

func (m *chatModelWithModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return m.ChatModel.Stream(ctx, input, append([]model.Option{model.WithModel(m.name)}, opts...)...)
}

func (m *einoChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	message, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderFromArray([]*schema.Message{message}), nil
}

func (m *chatModelWithModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	inner, ok := m.ChatModel.(model.ToolCallingChatModel)
	if !ok {
		return nil, fmt.Errorf("underlying ChatModel does not support WithTools")
	}
	cloned, err := inner.WithTools(tools)
	if err != nil {
		return nil, err
	}
	if em, ok := cloned.(*einoChatModel); ok {
		return &chatModelWithModel{ChatModel: em, name: m.name}, nil
	}
	return cloned, nil
}

func (m *einoChatModel) BindTools(tools []*schema.ToolInfo) error {
	m.tools = tools
	return nil
}

func (m *einoChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	clone := &einoChatModel{
		llm:   m.llm,
		tools: append([]*schema.ToolInfo(nil), tools...),
	}
	return clone, nil
}

var _ model.ChatModel = (*einoChatModel)(nil)
