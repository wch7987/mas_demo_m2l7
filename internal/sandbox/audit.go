package sandbox

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/tool"
	"github.com/zeromicro/go-zero/core/logx"
)

// NewAuditHandler 构造只读审计 Hook，对应课程的 after_tool_call / 观测类 Hook：
// 在工具开始（OnStart）与结束（OnEnd）时打印工具名、入参与结果。
//
// 关键区别：callbacks 只能观测（observe），不能改写或阻断工具入参；
// 真正的拦截/重写必须靠实现 tool.InvokableTool 的装饰器（见 secured_tool.go）。
// 两者配合，恰好对应课程的 before_tool_call（装饰器拦截）+ after_tool_call（callbacks 观测）。
func NewAuditHandler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if info != nil && info.Component == components.ComponentOfTool {
				if ci := tool.ConvCallbackInput(input); ci != nil {
					logx.Infof("[audit] 工具开始 tool=%s 入参=%s", info.Name, ci.ArgumentsInJSON)
				}
			}
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if info != nil && info.Component == components.ComponentOfTool {
				if co := tool.ConvCallbackOutput(output); co != nil {
					logx.Infof("[audit] 工具结束 tool=%s 结果=%s", info.Name, co.Response)
				}
			}
			return ctx
		}).
		Build()
}
