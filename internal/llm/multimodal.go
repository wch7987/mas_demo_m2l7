package llm

import (
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

// NormalizeMultimodalToolResult converts CrewAI-style tool image payloads into
// multimodal user messages for DashScope.
func NormalizeMultimodalToolResult(messages []APIMessage) ([]APIMessage, bool) {
	out := make([]APIMessage, 0, len(messages))
	flag := false
	pendingImages := make([]string, 0)

	for _, msg := range messages {
		role := msg.Role
		contentStr, _ := decodeContentString(msg.Content)

		if role == "tool" && strings.Contains(contentStr, "data:image/") && strings.Contains(contentStr, ";base64,") {
			dataURLStart := strings.Index(contentStr, "data:image/")
			prefix := contentStr[:dataURLStart]
			dataURL := contentStr[dataURLStart:]
			newMsg := msg
			if prefix != "" {
				newMsg.Content = encodeStringContent(prefix + "图片内容已加载")
			} else {
				newMsg.Content = encodeStringContent("图片内容已加载")
			}
			out = append(out, newMsg)
			pendingImages = append(pendingImages, dataURL)
			flag = true
			logx.Info("normalized_multimodal_tool_result: function calling mode, base64 in tool message")
			continue
		}

		if len(pendingImages) > 0 && role == "user" {
			parts := []MessageContentPart{{Type: "text", Text: contentStr}}
			for _, u := range pendingImages {
				parts = append(parts, MessageContentPart{
					Type:     "image_url",
					ImageURL: &struct{ URL string `json:"url"` }{URL: u},
				})
			}
			out = append(out, APIMessage{
				Role:    "user",
				Content: encodeContentParts(parts),
			})
			pendingImages = nil
			logx.Infof("normalized_multimodal_tool_result: injected %d image(s) into user message", len(parts)-1)
			continue
		}

		if role == "assistant" && contentStr != "" {
			if strings.Contains(contentStr, "Add image to content Local") && strings.Contains(contentStr, "data:image/") && strings.Contains(contentStr, ";base64,") {
				dataURLStart := strings.Index(contentStr, "data:image/")
				dataURL := contentStr[dataURLStart:]
				text := contentStr[:dataURLStart] + "图片内容已加载"
				out = append(out, APIMessage{
					Role: "user",
					Content: encodeContentParts([]MessageContentPart{
						{Type: "text", Text: text},
						{Type: "image_url", ImageURL: &struct{ URL string `json:"url"` }{URL: dataURL}},
					}),
				})
				flag = true
				logx.Info("normalized_multimodal_tool_result: ReAct mode, base64 in assistant message")
				continue
			}
			if strings.Contains(contentStr, "Add image to content Local") && strings.Contains(contentStr, "Observation: http") {
				obsStart := strings.Index(contentStr, "Observation: http")
				httpStart := strings.Index(contentStr[obsStart:], "http") + obsStart
				imageURL := contentStr[httpStart:]
				text := contentStr[:obsStart] + "图片内容已加载"
				out = append(out, APIMessage{
					Role: "user",
					Content: encodeContentParts([]MessageContentPart{
						{Type: "text", Text: text},
						{Type: "image", Image: imageURL},
					}),
				})
				flag = true
				continue
			}
		}

		out = append(out, msg)
	}

	if len(pendingImages) > 0 {
		parts := []MessageContentPart{{Type: "text", Text: "请分析上面工具返回的图片内容。"}}
		for _, u := range pendingImages {
			parts = append(parts, MessageContentPart{
				Type:     "image_url",
				ImageURL: &struct{ URL string `json:"url"` }{URL: u},
			})
		}
		out = append(out, APIMessage{
			Role:    "user",
			Content: encodeContentParts(parts),
		})
		logx.Infof("normalized_multimodal_tool_result: flushed %d pending image(s)", len(pendingImages))
	}

	return out, flag
}

// ValidateMessages checks message roles and required fields.
func ValidateMessages(messages []APIMessage) error {
	validRoles := map[string]struct{}{
		"system": {}, "user": {}, "assistant": {}, "tool": {},
	}
	for i, msg := range messages {
		if _, ok := validRoles[msg.Role]; !ok {
			return &ValidationError{Index: i, Message: "缺少或无效的 role"}
		}
		if msg.Role == "tool" {
			if msg.ToolCallID == "" || len(msg.Content) == 0 {
				return &ValidationError{Index: i, Message: "tool 消息缺少 tool_call_id/content"}
			}
			continue
		}
		if len(msg.Content) == 0 && len(msg.ToolCalls) == 0 {
			return &ValidationError{Index: i, Message: "缺少 content 且无 tool_calls"}
		}
	}
	return nil
}

// ValidationError indicates an invalid message at a given index.
type ValidationError struct {
	Index   int
	Message string
}

func (e *ValidationError) Error() string {
	return "消息 " + itoa(e.Index) + " " + e.Message
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
