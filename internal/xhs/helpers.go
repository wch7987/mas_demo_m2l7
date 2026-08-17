package xhs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	imageutil "github.com/cms/mas_demo_m2l7/internal/image"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	defaultMaxIter = 3
	toolFirstExtra = "硬性要求：第一次回复必须直接调用交付工具，禁止任何自然语言、思考过程、Markdown 或 JSON 文本。"
)

func drainEvents(iter *adk.AsyncIterator[*adk.AgentEvent]) error {
	for {
		event, ok := iter.Next()
		if !ok {
			return nil
		}
		if event.Err != nil {
			return event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		msg, err := event.Output.MessageOutput.GetMessage()
		if err != nil || msg == nil {
			continue
		}
		if len(msg.ToolCalls) > 0 {
			logx.Infof("xhs agent=%s tool=%s", event.AgentName, msg.ToolCalls[0].Function.Name)
		}
	}
}

func multimodalAlbum(text string, images []ImageInput) (*schema.Message, error) {
	parts := make([]schema.ChatMessagePart, 0, 1+len(images)*2)
	parts = append(parts, schema.ChatMessagePart{Type: schema.ChatMessagePartTypeText, Text: text})
	for _, img := range images {
		dataURL, err := imageutil.FileToDataURL(img.LocalPath)
		if err != nil {
			return nil, err
		}
		parts = append(parts,
			schema.ChatMessagePart{
				Type:     schema.ChatMessagePartTypeImageURL,
				ImageURL: &schema.ChatMessageImageURL{URL: dataURL},
			},
			schema.ChatMessagePart{
				Type: schema.ChatMessagePartTypeText,
				Text: fmt.Sprintf("以上图片 image_id=%s file_name=%s", img.ImageID, img.FileName),
			},
		)
	}
	return &schema.Message{Role: schema.User, MultiContent: parts}, nil
}

func sessionInput(keys ...string) adk.GenModelInput {
	return func(ctx context.Context, inst string, _ *adk.AgentInput) ([]*schema.Message, error) {
		var b []byte
		b = append(b, []byte("根据下列结构化输入完成你的职责。不要依赖对话历史。第一次回复必须直接调用交付工具。\n")...)
		for _, key := range keys {
			v, ok := adk.GetSessionValue(ctx, key)
			if !ok {
				return nil, fmt.Errorf("Session 缺少上游契约 %s", key)
			}
			raw, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("序列化 %s 失败: %w", key, err)
			}
			b = append(b, []byte("\n## "+key+"\n")...)
			b = append(b, raw...)
			b = append(b, '\n')
		}
		return []*schema.Message{schema.SystemMessage(inst), schema.UserMessage(string(b))}, nil
	}
}

func clampMaxIter(n int) int {
	if n <= 0 {
		return defaultMaxIter
	}
	return n
}
