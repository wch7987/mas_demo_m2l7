package xhs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type editBatchSubmit struct {
	ImagesEditPlan EditList `json:"images_edit_plan" jsonschema:"description=每张输入图片一条编辑方案，必须是 JSON 数组（不要再包一层字符串），顺序与输入一致"`
	Summary        string   `json:"summary" jsonschema:"description=不超过50字的整体编辑方向总结"`
}

func runEditPhase(ctx context.Context, cm model.ChatModel, maxIter int, idea string, images []ImageInput, visual *VisualBatchReport, onProgress ProgressFunc) (*ImageEditBatchReport, error) {
	emit(onProgress, ProgressEvent{
		Type:    "task_start",
		Task:    "edit",
		Agent:   imageEditorName,
		Message: fmt.Sprintf("%s 开始修图方案 %d 张图", imageEditorRole, len(images)),
	})

	state := &editBatchSubmit{}
	submit, err := utils.InferTool(
		"submit_edit_batch",
		"一次性提交全部图片的小红书内置编辑方案。images_edit_plan 必须是 JSON 数组对象，不要把数组编码成字符串。条数必须等于输入图片数。",
		func(_ context.Context, out *editBatchSubmit) (string, error) {
			*state = *out
			return "编辑方案批次已提交", nil
		},
	)
	if err != nil {
		return nil, err
	}

	visualJSON, _ := json.Marshal(visual)
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        imageEditorName,
		Description: imageEditorRole,
		Instruction: instruction(imageEditorRole, imageEditorGoal, imageEditorBackstory, toolFirstExtra),
		Model:       cm,
		GenModelInput: func(_ context.Context, inst string, _ *adk.AgentInput) ([]*schema.Message, error) {
			prompt := fmt.Sprintf(
				"根据用户意图和视觉分析，为下面全部图片给出小红书内置编辑方案。\n用户意图：%s\n共 %d 张图。\n视觉分析：%s\n请观察每张图，然后调用 submit_edit_batch。不需要编辑的字段写「不需要编辑」。",
				idea, len(images), string(visualJSON),
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
			ReturnDirectly:  map[string]bool{"submit_edit_batch": true},
		},
	})
	if err != nil {
		return nil, err
	}

	iter := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent}).Query(ctx, "开始生成修图方案")
	if err := drainEvents(iter); err != nil {
		return nil, err
	}

	batch, err := alignEditBatch(images, state)
	if err != nil {
		return nil, err
	}
	for i, p := range batch.ImagesEditPlan {
		emit(onProgress, ProgressEvent{
			Type:    "task_done",
			Task:    "edit:" + p.ImageID,
			Agent:   imageEditorName,
			Message: imageEditorRole + " 完成 " + p.FileName,
			Content: truncateRunes(p.OverallEditStrategy, 4000),
			Step:    i + 1,
		})
	}
	return batch, nil
}

func alignEditBatch(images []ImageInput, submitted *editBatchSubmit) (*ImageEditBatchReport, error) {
	if submitted == nil || len(submitted.ImagesEditPlan) != len(images) {
		n := 0
		if submitted != nil {
			n = len(submitted.ImagesEditPlan)
		}
		return nil, fmt.Errorf("模型未完整提交修图方案：得到 %d 条，期望 %d 条", n, len(images))
	}
	byID := make(map[string]ImageEditPlan, len(submitted.ImagesEditPlan))
	for _, p := range submitted.ImagesEditPlan {
		if p.ImageID != "" {
			byID[p.ImageID] = p
		}
	}
	out := make([]ImageEditPlan, 0, len(images))
	for i, img := range images {
		p, ok := byID[img.ImageID]
		if !ok {
			p = submitted.ImagesEditPlan[i]
		}
		p.ImageID = img.ImageID
		p.FileName = img.FileName
		if p.OverallEditStrategy == "" {
			return nil, fmt.Errorf("修图方案缺少整体策略: %s", img.ImageID)
		}
		out = append(out, p)
	}
	return &ImageEditBatchReport{ImagesEditPlan: out, Summary: submitted.Summary}, nil
}
