package todo

import (
	"context"
	"os"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cms/mas_demo_m2l7/internal/sandbox"
)

// WriteFileArgs 是 write_file 工具的入参。
//
// 注意：原课程（Python/CrewAI）用了 file_path / filename 两种字段名，容易让装饰器难以统一拦截。
// Go 版本有意把路径字段统一成 "path"，安全装饰器（sandbox.NewSecuredFileTool）只认 "path" 一个字段，
// 是有意为之的改进：字段语义单一，拦截与重写逻辑无需关心工具内部用的是哪个名字。
type WriteFileArgs struct {
	Path    string `json:"path" jsonschema:"description=要写入的文件名，例如 TODO.md（只给文件名，不要带目录）"`
	Content string `json:"content" jsonschema:"description=要写入文件的完整内容"`
}

// ReadFileArgs 是 read_file 工具的入参，路径字段同样统一为 "path"。
type ReadFileArgs struct {
	Path string `json:"path" jsonschema:"description=要读取的文件名，例如 TODO.md"`
}

// NewFileTools 构造并返回已被安全装饰器包裹的文件工具列表。
//
// 约定：下标 0 必须是 write_file（main 的第 5 轮模拟注入演示会直接取 tools[0] 调用），下标 1 是 read_file。
// 两个工具都用 sandbox.NewSecuredFileTool(..., "path") 装饰，因此 Agent 看到的工具描述与原始一致，
// 但实际执行前路径会被透明重写到 ./workspace/{uid}/，并阻断 ../ 穿越。
func NewFileTools() ([]tool.BaseTool, error) {
	writeTool, err := utils.InferTool("write_file", "把内容写入指定文件（会覆盖原内容）。", writeFile)
	if err != nil {
		return nil, err
	}
	readTool, err := utils.InferTool("read_file", "读取指定文件的全部内容。", readFile)
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{
		sandbox.NewSecuredFileTool(writeTool, "path"), // 下标 0 = write
		sandbox.NewSecuredFileTool(readTool, "path"),  // 下标 1 = read
	}, nil
}

// writeFile 覆盖写入文件全量内容。路径在进入这里之前已被装饰器重写为工作空间内的绝对路径。
func writeFile(_ context.Context, in *WriteFileArgs) (string, error) {
	if err := os.WriteFile(in.Path, []byte(in.Content), 0o644); err != nil {
		return "", err
	}
	return "已写入文件：" + in.Path, nil
}

// readFile 读取文件全量内容。
// 文件不存在不是错误：返回友好提示字符串（nil error），让模型据此理解“暂无待办”。
func readFile(_ context.Context, in *ReadFileArgs) (string, error) {
	data, err := os.ReadFile(in.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return "文件不存在，暂无待办。", nil
		}
		return "", err
	}
	return string(data), nil
}
