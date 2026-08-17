# mas_demo_m2l7

Go + Eino 实现的小红书爆款笔记主链路（对应 Python `crewai_mas_demo_m2l7` 实战）。HTTP 用 SSE 推阶段进度；CLI 同步打印报告。

第一期范围：HTTP + CLI + 三阶段编排 + 图片压缩/临时目录。不含鉴权、限流、Prometheus、K8s、数据库。

## 流程

1. 保存并压缩本地图片（长边 1024，JPEG quality 85，最多 20 张）
2. 视觉分析：一次 VL 请求看完全部图，工具提交整批结果
3. 修图方案：一次 VL 请求看完全部图，工具提交整批结果
4. SequentialAgent：策略 → 文案 → SEO（各一次交付工具；Session 契约，不靠 History 文本）

## 环境变量

与 `mas_demo` 相同，不要另配一套：

- `QWEN_API_KEY` 或 `DASHSCOPE_API_KEY`（必填）
- 可选：`DASHSCOPE_BASE_URL`、`QWEN_MODEL`、`QWEN_IMAGE_MODEL`、`QWEN_REGION`

视觉/修图默认 `qwen3-vl-plus`（`app/etc/main.yaml` 的 `LLM.ImageModel`，CLI 在未设 `QWEN_IMAGE_MODEL` 时同样默认该值）。文本阶段用 `LLM.Model`。

## CLI

```bash
export QWEN_API_KEY=...
go run ./cmd/xhs -idea "想卖这个墨绿色马克杯，主打独居女生，强调氛围感" \
  -images /path/to/a.jpg,/path/to/b.jpg
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

## 测试（无 Key）

```bash
go test ./...
go build ./...
```
