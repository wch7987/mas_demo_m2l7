package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cms/mas_demo_m2l7/internal/llm"
	"github.com/cms/mas_demo_m2l7/internal/sandbox"
	"github.com/cms/mas_demo_m2l7/internal/todo"
)

func main() {
	cfg, err := llm.ConfigFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置错误: %v\n请设置环境变量 QWEN_API_KEY 或 DASHSCOPE_API_KEY 后重试。\n", err)
		os.Exit(1)
	}

	cm, err := llm.NewEinoChatModel(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化 LLM 失败: %v\n", err)
		os.Exit(1)
	}

	// 固定 user_id，演示请求级多租户隔离：所有文件都会落在 ./workspace/1234567890/ 下。
	ctx := sandbox.WithUserID(context.Background(), "1234567890")
	svc := todo.NewService(cm, 8)

	// 前 4 轮都是正常业务对话，全部走真实 Agent（svc.Run）。
	// 跨轮状态不靠对话历史，而是靠 ./workspace/1234567890/TODO.md 文件持久化：
	// 第 4 轮把「买牛奶」也标记完成后，清单里「买牛奶」「写周报」都应为已完成。
	rounds := []string{
		"帮我加三条待办：买牛奶、写周报、预约牙医。",
		"看看我现在有哪些待办。",
		"把'写周报'标记为已完成。",
		"把'买牛奶'也标记为已完成。",
	}

	for i, round := range rounds {
		fmt.Printf("\n========== 第 %d 轮 ==========\n", i+1)
		fmt.Printf("用户：%s\n", round)
		reply, err := svc.Run(ctx, round)
		if err != nil {
			fmt.Fprintf(os.Stderr, "第 %d 轮执行失败: %v\n", i+1, err)
			os.Exit(1)
		}
		fmt.Printf("助手：%s\n", reply)
	}

	// 第 5 轮：模拟注入攻击——目录穿越被拦截。这一轮仍属于“对话流”的一部分，但刻意不走 Agent。
	//
	// 原因：装饰器拦截的是“实际传给工具的 path”。真实 Agent 下 path 由 LLM 决定，而 backstory
	// 已把 LLM 约束成只用文件名 "TODO.md"、绝不带目录前缀，所以正常自然语言对话里 LLM 不会吐出 ../
	// 穿越路径，拦截无从稳定触发。为了把“安全装饰器确实挡住了穿越”演示出来，这里直接以工具入参的
	// 形式注入一个穿越路径，模拟模型被 prompt 注入 / 越狱后吐出恶意路径的最坏情形。
	fmt.Printf("\n========== 第 5 轮（模拟注入攻击：目录穿越被拦截） ==========\n")
	fmt.Println("被注入的恶意请求：把清单导出到 ../../../../etc/passwd")

	tools, err := todo.NewFileTools()
	if err != nil {
		fmt.Fprintf(os.Stderr, "构造文件工具失败: %v\n", err)
		os.Exit(1)
	}

	// tools[0] = write_file。content 用“标记完成后的清单”文本当业务外壳，
	// 模拟攻击者想把用户真实清单顺手导出（exfiltrate）到工作空间之外的 /etc/passwd。
	writeTool := tools[0].(tool.InvokableTool)
	out, err := writeTool.InvokableRun(ctx, `{"path":"../../../../etc/passwd","content":"- [x] 买牛奶\n- [x] 写周报\n- [ ] 预约牙医\n"}`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "注入演示执行异常: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("拦截返回：", out)

	// 佐证工作空间安全：注入被挡下后，用合法入参 {"path":"TODO.md"} 读回真实清单。
	// tools[1] = read_file。读回的内容应当仍是前 4 轮留下的正常待办（买牛奶、写周报都已打勾），
	// 证明这次注入既没写出工作空间、也没污染/覆盖工作空间内的真实 TODO.md。
	readTool := tools[1].(tool.InvokableTool)
	back, err := readTool.InvokableRun(ctx, `{"path":"TODO.md"}`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读回 TODO.md 异常: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("读回 ./workspace/1234567890/TODO.md（未被注入污染）：\n%s\n", back)
}
