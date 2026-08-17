package xhs

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

func runContentPhase(ctx context.Context, cm model.ChatModel, maxIter int, idea string, visual *VisualBatchReport, edit *ImageEditBatchReport, onProgress ProgressFunc) (*ContentResult, error) {
	state := &ContentResult{}
	submitStrategy, submitCopy, submitSEO, err := newContentSubmitTools(state)
	if err != nil {
		return nil, err
	}

	strategist, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          strategistName,
		Description:   strategistRole + "，只交付策略简报",
		Instruction:   instruction(strategistRole, strategistGoal, strategistBackstory, toolFirstExtra),
		Model:         cm,
		GenModelInput: sessionInput(sessionIdeaText, sessionVisualReport, sessionEditReport),
		MaxIterations: clampMaxIter(maxIter),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{Tools: []tool.BaseTool{submitStrategy}},
			ReturnDirectly:  map[string]bool{"submit_content_strategy_brief": true},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("strategist: %w", err)
	}

	writer, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          writerName,
		Description:   writerRole + "，只交付文案",
		Instruction:   instruction(writerRole, writerGoal, writerBackstory, toolFirstExtra),
		Model:         cm,
		GenModelInput: sessionInput(sessionIdeaText, sessionVisualReport, sessionStrategyBrief),
		MaxIterations: clampMaxIter(maxIter),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{Tools: []tool.BaseTool{submitCopy}},
			ReturnDirectly:  map[string]bool{"submit_copywriting": true},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("writer: %w", err)
	}

	seo, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          seoName,
		Description:   seoRole + "，只交付 SEO 报告",
		Instruction:   instruction(seoRole, seoGoal, seoBackstory, toolFirstExtra),
		Model:         cm,
		GenModelInput: sessionInput(sessionIdeaText, sessionStrategyBrief, sessionCopywriting),
		MaxIterations: clampMaxIter(maxIter),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{Tools: []tool.BaseTool{submitSEO}},
			ReturnDirectly:  map[string]bool{"submit_seo_report": true},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("seo: %w", err)
	}

	seq, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "xhs_content",
		Description: "小红书笔记内容流水线：策略 → 文案 → SEO",
		SubAgents:   []adk.Agent{strategist, writer, seo},
	})
	if err != nil {
		return nil, err
	}

	emit(onProgress, ProgressEvent{
		Type:    "task_start",
		Task:    "content",
		Agent:   strategistName,
		Message: "内容阶段开始：策略 → 文案 → SEO",
	})

	iter := adk.NewRunner(ctx, adk.RunnerConfig{Agent: seq}).Query(ctx, "开始执行小红书笔记内容流水线",
		adk.WithSessionValues(map[string]any{
			sessionIdeaText:     idea,
			sessionVisualReport: visual,
			sessionEditReport:   edit,
		}),
	)
	if err := drainEvents(iter); err != nil {
		return nil, err
	}
	if state.SEO == nil {
		return state, fmt.Errorf("SEO 步骤未提交结构化报告，请检查模型是否调用了 submit_seo_report")
	}

	emit(onProgress, ProgressEvent{
		Type:    "task_done",
		Task:    "content",
		Agent:   seoName,
		Message: "内容阶段完成",
		Content: truncateRunes(state.SEO.OptimizedTitle, 4000),
	})
	return state, nil
}

func newContentSubmitTools(state *ContentResult) (strategy, copy, seo tool.BaseTool, err error) {
	strategy, err = utils.InferTool(
		"submit_content_strategy_brief",
		"提交内容策略简报。必须填齐所有字段，这是本步唯一合法的最终交付。",
		func(ctx context.Context, brief *ContentStrategyBrief) (string, error) {
			adk.AddSessionValue(ctx, sessionStrategyBrief, brief)
			state.Strategy = brief
			return "策略简报已写入 Session，流水线可进入文案步骤", nil
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	copy, err = utils.InferTool(
		"submit_copywriting",
		"提交小红书笔记文案。必须填齐标题、正文、图片顺序和钩子句。",
		func(ctx context.Context, out *CopywritingOutput) (string, error) {
			adk.AddSessionValue(ctx, sessionCopywriting, out)
			state.Copywriting = out
			return "文案已写入 Session，流水线可进入 SEO 步骤", nil
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	seo, err = utils.InferTool(
		"submit_seo_report",
		"提交 SEO 优化后的笔记。必须填齐所有字段，标签 5-8 个。",
		func(ctx context.Context, report *SEOOptimizedNote) (string, error) {
			adk.AddSessionValue(ctx, sessionSEOReport, report)
			state.SEO = report
			return "SEO 报告已写入 Session，流水线结束", nil
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	return strategy, copy, seo, nil
}
