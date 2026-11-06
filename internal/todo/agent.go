package todo

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/cms/mas_demo_m2l7/internal/sandbox"
)

const (
	todoAgentName  = "todo_manager"
	todoAgentDesc  = "待办清单管理助手"
	defaultMaxIter = 8
)

// todoBackstory 是待办清单管理助手的系统指令（Instruction）。
// 它定义助手的身份与职责，规定 TODO.md 的行格式，以及“先读全量再覆盖写回”的工具使用纪律。
const todoBackstory = `你是一名「待办清单管理助手」。你帮用户维护一个 TODO.md 待办清单文件，
支持以下操作：新增待办、列出待办、把某条待办标记为已完成、删除待办。

## 待办存储：TODO.md
所有待办都保存在名为 TODO.md 的文件里，一行一条，使用标准 markdown 复选框格式：
- [ ] 表示未完成的待办，例如：- [ ] 买牛奶
- [x] 表示已完成的待办，例如：- [x] 写周报

## 工具使用纪律
1. 文件名只给 "TODO.md"，绝不要带任何目录前缀（系统会自动隔离到你的专属工作空间）。
2. 新增或修改任何待办前，必须先用 read_file 读取 TODO.md 的全量内容，在内存里合并/修改后，再用 write_file 写回全量内容。
   write_file 是覆盖写入，如果不先读全量再写回，会丢失其它待办。
3. 查询待办直接用 read_file 读取 TODO.md。
4. 完成文件操作后，用友好的自然语言向用户说明你做了什么，可以用 markdown 列表展示当前的待办清单。`

// NewTodoAgent 构造待办清单管理助手 Agent。
// 工具是已被安全装饰器包裹的 write_file/read_file；不设 ReturnDirectly，
// 因为调用完文件工具后还需要模型组织一段自然语言回复给用户。
func NewTodoAgent(ctx context.Context, cm model.ChatModel, maxIter int) (adk.Agent, error) {
	tools, err := NewFileTools()
	if err != nil {
		return nil, err
	}
	if maxIter <= 0 {
		maxIter = defaultMaxIter
	}
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          todoAgentName,
		Description:   todoAgentDesc,
		Instruction:   todoBackstory,
		Model:         cm,
		MaxIterations: maxIter,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{Tools: tools},
		},
	})
}

// Service 持有 ChatModel 与最大迭代次数，提供按轮执行的便捷封装。
// 跨轮状态不依赖对话历史，而是靠 ./workspace/{uid}/TODO.md 文件持久化。
type Service struct {
	cm      model.ChatModel
	maxIter int
}

// NewService 创建 Service。
func NewService(cm model.ChatModel, maxIter int) *Service {
	return &Service{cm: cm, maxIter: maxIter}
}

// Run 执行一轮对话：构造 Agent，用 Runner.Query 跑一次，挂上观测型审计 Handler，
// 最后返回模型的自然语言最终回复。
func (s *Service) Run(ctx context.Context, userInput string) (string, error) {
	agent, err := NewTodoAgent(ctx, s.cm, s.maxIter)
	if err != nil {
		return "", err
	}
	iter := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent}).
		Query(ctx, userInput, adk.WithCallbacks(sandbox.NewAuditHandler()))
	return collectFinalText(iter)
}

// collectFinalText 消费事件流，返回最后一条「助手发出、无工具调用、有正文」的消息内容，
// 即模型组织好的自然语言最终回复。
func collectFinalText(iter *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	final := ""
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		msg, err := event.Output.MessageOutput.GetMessage()
		if err != nil || msg == nil {
			continue
		}
		if msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 && msg.Content != "" {
			final = msg.Content
		}
	}
	return final, nil
}
