package xhs

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type visualBatchSubmit struct {
	ImagesVisual []ImageVisualAnalysis `json:"images_visual" jsonschema:"description=每张输入图片一条视觉分析，顺序与输入一致，条数必须相同"`
	Summary      string                `json:"summary" jsonschema:"description=不超过50字的整体视觉基调总结"`
}

func runVisualPhase(ctx context.Context, cm model.ChatModel, maxIter int, idea string, images []ImageInput, onProgress ProgressFunc) (*VisualBatchReport, error) {
	emit(onProgress, ProgressEvent{
		Type:    "task_start",
		Task:    "visual",
		Agent:   visualAnalystName,
		Message: fmt.Sprintf("%s 开始分析 %d 张图", visualAnalystRole, len(images)),
	})

	state := &visualBatchSubmit{}
	submit, err := utils.InferTool(
		"submit_visual_batch",
		"一次性提交全部图片的视觉分析。images_visual 条数必须等于输入图片数，这是本步唯一合法的最终交付。",
		func(_ context.Context, out *visualBatchSubmit) (string, error) {
			*state = *out
			return "视觉分析批次已提交", nil
		},
	)
	if err != nil {
		return nil, err
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        visualAnalystName,
		Description: visualAnalystRole,
		Instruction: instruction(visualAnalystRole, visualAnalystGoal, visualAnalystBackstory, toolFirstExtra),
		Model:       cm,
		GenModelInput: func(_ context.Context, inst string, _ *adk.AgentInput) ([]*schema.Message, error) {
			prompt := fmt.Sprintf(
				"根据用户创作意图，对下面全部图片做结构化视觉分析。\n用户意图：%s\n共 %d 张图，请按出现顺序分析每一张，然后调用 submit_visual_batch。不要编造看不到的内容。",
				idea, len(images),
			)
			user, err := multimodalAlbum(prompt, images)
			if err != nil {
				return nil, err
			}
			return []*schema.Message{schema.SystemMessage(inst), user}, nil
		},
		MaxIterations: clampMaxIter(maxIter),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{Tools: []tool.BaseTool{submit}},
			ReturnDirectly:  map[string]bool{"submit_visual_batch": true},
		},
	})
	if err != nil {
		return nil, err
	}

	iter := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent}).Query(ctx, "开始视觉分析")
	if err := drainEvents(iter); err != nil {
		return nil, err
	}

	batch, err := alignVisualBatch(idea, images, state)
	if err != nil {
		return nil, err
	}

	for i, v := range batch.ImagesVisual {
		emit(onProgress, ProgressEvent{
			Type:    "task_done",
			Task:    "visual:" + v.ImageID,
			Agent:   visualAnalystName,
			Message: visualAnalystRole + " 完成 " + v.FileName,
			Content: truncateRunes(v.HighlightFeature+" / "+v.AtmosphereVibe, 4000),
			Step:    i + 1,
		})
	}
	return batch, nil
}

func alignVisualBatch(idea string, images []ImageInput, submitted *visualBatchSubmit) (*VisualBatchReport, error) {
	if submitted == nil || len(submitted.ImagesVisual) != len(images) {
		n := 0
		if submitted != nil {
			n = len(submitted.ImagesVisual)
		}
		return nil, fmt.Errorf("模型未完整提交视觉分析：得到 %d 条，期望 %d 条", n, len(images))
	}
	byID := make(map[string]ImageVisualAnalysis, len(submitted.ImagesVisual))
	for _, v := range submitted.ImagesVisual {
		if v.ImageID != "" {
			byID[v.ImageID] = v
		}
	}
	out := make([]ImageVisualAnalysis, 0, len(images))
	for i, img := range images {
		v, ok := byID[img.ImageID]
		if !ok {
			v = submitted.ImagesVisual[i]
		}
		v.ImageID = img.ImageID
		v.FileName = img.FileName
		if v.SubjectDescription == "" {
			return nil, fmt.Errorf("视觉分析缺少主体描述: %s", img.ImageID)
		}
		out = append(out, v)
	}
	return &VisualBatchReport{
		UserRawIntent: idea,
		ImagesVisual:  out,
		Summary:       submitted.Summary,
	}, nil
}
