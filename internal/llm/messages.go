package llm

import (
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/schema"
)

func toolInfosToAPI(tools []*schema.ToolInfo) ([]APITool, error) {
	if len(tools) == 0 {
		return nil, nil
	}
	out := make([]APITool, 0, len(tools))
	for _, toolInfo := range tools {
		if toolInfo == nil {
			continue
		}
		apiTool, err := toolInfoToAPI(toolInfo)
		if err != nil {
			return nil, err
		}
		out = append(out, apiTool)
	}
	return out, nil
}

func toolInfoToAPI(toolInfo *schema.ToolInfo) (APITool, error) {
	params := map[string]any{"type": "object", "properties": map[string]any{}}
	if toolInfo.ParamsOneOf != nil {
		jsonSchema, err := toolInfo.ParamsOneOf.ToJSONSchema()
		if err != nil {
			return APITool{}, fmt.Errorf("tool %s schema: %w", toolInfo.Name, err)
		}
		if jsonSchema != nil {
			raw, err := json.Marshal(jsonSchema)
			if err != nil {
				return APITool{}, err
			}
			if err := json.Unmarshal(raw, &params); err != nil {
				return APITool{}, err
			}
		}
	}
	return APITool{
		Type: "function",
		Function: APIFunctionDef{
			Name:        toolInfo.Name,
			Description: toolInfo.Desc,
			Parameters:  params,
		},
	}, nil
}

func schemaMessagesToAPI(input []*schema.Message) ([]APIMessage, error) {
	out := make([]APIMessage, 0, len(input))
	for _, message := range input {
		if message == nil {
			continue
		}
		apiMessage, err := schemaMessageToAPI(message)
		if err != nil {
			return nil, err
		}
		out = append(out, apiMessage)
	}
	if len(out) == 0 {
		return []APIMessage{{Role: "user", Content: encodeStringContent("")}}, nil
	}
	return out, nil
}

func schemaMessageToAPI(message *schema.Message) (APIMessage, error) {
	role := string(message.Role)
	if role == "" {
		role = "user"
	}

	apiMessage := APIMessage{
		Role: role,
		Name: message.Name,
	}

	if len(message.MultiContent) > 0 {
		parts := make([]MessageContentPart, 0, len(message.MultiContent))
		for _, part := range message.MultiContent {
			switch part.Type {
			case schema.ChatMessagePartTypeText:
				parts = append(parts, MessageContentPart{Type: "text", Text: part.Text})
			case schema.ChatMessagePartTypeImageURL:
				if part.ImageURL != nil {
					parts = append(parts, MessageContentPart{
						Type: "image_url",
						ImageURL: &struct {
							URL string `json:"url"`
						}{URL: part.ImageURL.URL},
					})
				}
			}
		}
		apiMessage.Content = encodeContentParts(parts)
	} else {
		apiMessage.Content = encodeStringContent(message.Content)
	}

	if message.Role == schema.Tool {
		apiMessage.Role = "tool"
		apiMessage.ToolCallID = message.ToolCallID
	}

	if len(message.ToolCalls) > 0 {
		apiMessage.ToolCalls = make([]APIToolCall, 0, len(message.ToolCalls))
		for _, toolCall := range message.ToolCalls {
			apiMessage.ToolCalls = append(apiMessage.ToolCalls, APIToolCall{
				ID:   toolCall.ID,
				Type: toolCall.Type,
				Function: APIFunctionCall{
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				},
			})
		}
	}

	return apiMessage, nil
}

func apiAssistantToSchema(assistant APIAssistantMessage) *schema.Message {
	toolCalls := make([]schema.ToolCall, 0, len(assistant.ToolCalls))
	for _, toolCall := range assistant.ToolCalls {
		callType := toolCall.Type
		if callType == "" {
			callType = "function"
		}
		toolCalls = append(toolCalls, schema.ToolCall{
			ID:   toolCall.ID,
			Type: callType,
			Function: schema.FunctionCall{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		})
	}
	return schema.AssistantMessage(assistant.Content, toolCalls)
}
