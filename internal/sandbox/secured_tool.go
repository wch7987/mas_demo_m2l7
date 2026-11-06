package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

// workspaceRoot 是所有用户隔离目录的根；每个用户的文件最终落在 ./workspace/{uid}/ 下。
const workspaceRoot = "./workspace"

// securedFileTool 是文件工具的装饰器（decorator），对应课程的 before_tool_call：
// 在真正执行内部工具前，把入参里的路径字段透明重写到 ./workspace/{uid}/，并阻断 ../ 穿越。
// Agent/模型对这层重写完全无感——它看到的工具 Info 与原始工具一致。
type securedFileTool struct {
	inner     tool.InvokableTool
	pathField string // 要被校验/重写的参数字段名，本项目统一为 "path"
}

var _ tool.InvokableTool = (*securedFileTool)(nil)

// NewSecuredFileTool 用安全装饰器包裹一个 InvokableTool，pathField 指定需要重写的路径字段。
func NewSecuredFileTool(inner tool.InvokableTool, pathField string) tool.InvokableTool {
	return &securedFileTool{inner: inner, pathField: pathField}
}

// Info 透传内部工具的描述，使模型对装饰层无感知。
func (s *securedFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return s.inner.Info(ctx)
}

// InvokableRun 是拦截核心（before_tool_call）。
//
// 铁律：所有阻断/错误分支一律返回 (说明字符串, nil)，绝不返回 Go error。
// 对齐课程语义——错误信息会作为工具结果回传给模型，由模型据此调整，而不是中断整条链路。
func (s *securedFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	logx.Infof("[secured] 工具调用 field=%s", s.pathField)
	logx.Infof("[secured] 工具输入 %s", argumentsInJSON)

	uid, ok := UserID(ctx)
	if !ok || uid == "" {
		return "缺少必要的上下文信息：user_id 为空", nil
	}

	base := filepath.Join(workspaceRoot, uid)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return fmt.Sprintf("创建工作空间失败：%s", err), nil
	}

	args := map[string]any{}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return fmt.Sprintf("工具入参解析失败：%s", err), nil
	}

	raw, _ := args[s.pathField].(string)
	if raw == "" {
		return "文件路径不能为空", nil
	}

	safe, err := SafePath(base, raw)
	if errors.Is(err, ErrPathTraversal) {
		return fmt.Sprintf("非法的路径：%s（路径超出工作空间范围）", raw), nil
	}
	if err != nil {
		return fmt.Sprintf("路径校验失败：%s", err), nil
	}

	args[s.pathField] = safe
	rewritten, err := json.Marshal(args)
	if err != nil {
		return fmt.Sprintf("工具入参重写失败：%s", err), nil
	}
	logx.Infof("[secured] 校验后的工具输入 %s", string(rewritten))

	return s.inner.InvokableRun(ctx, string(rewritten), opts...)
}
