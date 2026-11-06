# mas_demo_m2l7

Go + Eino 实现的小红书爆款笔记主链路（对应 Python `crewai_mas_demo_m2l7` 实战）。HTTP 用 SSE 推阶段进度；CLI 同步打印报告。

第一期范围：HTTP + CLI + 三阶段编排 + 图片压缩/临时目录。不含鉴权、限流、Prometheus、K8s、数据库。

> 本仓库包含两个独立的教学 demo：
> 1. **小红书爆款笔记主链路**（下方「流程 / CLI / HTTP」）——多模态 + 三阶段编排。
> 2. **Hook 拦截与多租户隔离（TODO 待办清单）**——见下方「Demo 2」章节，详细讲义在 [`internal/todo/README.md`](internal/todo/README.md)。

## 流程

1. 保存并压缩本地图片（长边 1024，JPEG quality 85，最多 20 张）
2. 视觉分析：一次 VL 请求看完全部图，工具提交整批结果
3. 修图方案：一次 VL 请求看完全部图，工具提交整批结果
4. SequentialAgent：策略 → 文案 → SEO（各一次交付工具；Session 契约，不靠 History 文本）

## 环境变量

与 `mas_demo` 相同，不要另配一套：

- `QWEN_API_KEY` 或 `DASHSCOPE_API_KEY`（必填）
- 可选：`DASHSCOPE_BASE_URL`、`QWEN_MODEL`、`QWEN_IMAGE_MODEL`、`QWEN_REGION`

视觉/修图与文本默认都用 `qwen3.7-plus`（与 mas_demo 同一套已开通模型；`qwen3.7-plus` 本身支持多模态）。若账号已开通 VL 专用模型，可设 `QWEN_IMAGE_MODEL` 或改 yaml 的 `LLM.ImageModel`。

## CLI

```bash
export QWEN_API_KEY=...
go run ./cmd/xhs -idea "想卖这个墨绿色马克杯，主打独居女生，强调氛围感" \
  -images /path/to/a.jpg,/path/to/b.jpg
  
curl -N -F 'idea_text=北京马拉松大正再次PB，感慨大正的努力，主打努力，没有白练的' \
  -F 'images=@/Users/cms/Desktop/dazheng.jpg' \
  http://127.0.0.1:8888/api/v1/xhs/notes/report 
```

## HTTP（SSE）

```bash
cd app
go run main.go -f etc/main.yaml

# -N 关闭缓冲
curl -N -F 'idea_text=想卖马克杯' \
  -F 'images=@/path/to/a.jpg' \
  -F 'images=@/path/to/b.jpg' \
  http://127.0.0.1:8888/api/v1/xhs/notes/report
```

事件：`start` → `task_start` / `task_done` → `done`（`content` 为完整报告）或 `error`。

一次完整跑可能 5–15 分钟。

## Demo 2：Hook 拦截与多租户隔离（TODO 待办清单）

课程「用 Hook 拦截实现安全控制与多租户隔离」的 Go / eino 实现，由早期的 crontab 定时任务 demo 改造而来——改成待办清单是为了消除「把任务记录进文件却从不执行」的语义歧义（待办清单的交付物就是文件内容本身，写进去即完成）。

一个「待办清单管理助手」Agent 用 `write_file` / `read_file` 维护 `TODO.md`，而底层所有文件读写都被一层安全机制**透明重定向**到按用户隔离的工作空间 `./workspace/{user_id}/`，并挡住 `../` 路径穿越。Agent 只给文件名 `TODO.md`，对重写毫无感知。

三大核心点（Python / CrewAI → Go / eino）：

| 课程（Python） | 本实现（Go / eino） | 落点 |
|---|---|---|
| `contextvars` 传 user_id | `context.Context` + `WithValue` | `internal/sandbox/context.go` |
| `@before_tool_call` 改写 / 阻断入参 | Tool 装饰器实现 `tool.InvokableTool` | `internal/sandbox/secured_tool.go` |
| `Path.resolve()` + `relative_to()` 防穿越 | `filepath.Abs` / `Join` / `Rel` 两阶段 | `internal/sandbox/safepath.go` |
| `@after_tool_call`（观测） | eino `callbacks.Handler` | `internal/sandbox/audit.go` |

目录：

- `internal/sandbox/`：安全核心，与业务无关、可复用（含无 Key 单测）
- `internal/todo/`：待办清单业务 Agent + 文件工具，详细讲义见 [`internal/todo/README.md`](internal/todo/README.md)
- `cmd/todo/`：CLI 入口

跑法：

```bash
# ① 安全核心单测（无需 Key，课程重点：路径穿越防御 + 多租户隔离）
go test ./internal/sandbox/...

# ② 端到端（需 Key）：四轮正常对话 + 第 5 轮模拟注入攻击拦截演示
export QWEN_API_KEY=...
go run ./cmd/todo
```

端到端先跑四轮正常业务对话——加三条待办 → 列出 → 把「写周报」标记完成 → 把「买牛奶」也标记完成——产物写到 `./workspace/1234567890/TODO.md`（第 4 轮后「买牛奶」「写周报」都已打勾）：

```
- [x] 买牛奶
- [x] 写周报
- [ ] 预约牙医
```

第 5 轮融入对话流，模拟模型被 prompt 注入 / 越狱后吐出恶意路径的最坏情形：直接以工具入参注入 `../../../../etc/passwd`，被安全装饰器拦截，打印「非法的路径：../../../../etc/passwd（路径超出工作空间范围）」；随后用合法入参读回 `TODO.md`，证明工作空间里的真实清单未被注入污染（仍是上面四轮留下的待办）。

## 测试（无 Key）

```bash
go test ./...
go build ./...
```
