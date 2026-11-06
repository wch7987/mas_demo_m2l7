# 用 Hook 拦截实现安全控制与多租户隔离（Go / eino 版）

本模块把课程「用 Hook 拦截实现安全控制与多租户隔离」的 Python/CrewAI 讲解，移植为 Go + [CloudWeGo eino](https://github.com/cloudwego/eino) 实现。业务场景是一个「待办清单管理助手」Agent：它帮用户维护一个 `TODO.md` 待办清单（新增待办、列出待办、标记某条为已完成、删除待办），而底层的文件读写被一层安全机制包裹，实现路径穿越防御与按用户隔离的工作空间。待办清单的交付物就是文件内容本身——「写进文件就是完成了」，不存在延迟执行的歧义。

## 主题

同一套工具（`write_file` / `read_file`）要服务多个用户，必须保证：

1. **多租户隔离**：用户 A 的文件绝不能落到用户 B 的目录，用户身份随请求传递、并发安全。
2. **安全控制**：恶意或失误的 `../../etc/passwd` 这类路径穿越必须被挡在工作空间之外。
3. **可观测**：每次工具调用的入参/结果都要能被审计记录。

课程在 Python 里用 CrewAI 的 `@before_tool_call` / `@after_tool_call` Hook + `contextvars` 完成；本实现用 eino 的「工具装饰器 + callbacks + `context.Context`」等价达成。

## 三大核心点：Python → Go 映射表

| 课程（Python / CrewAI） | 本实现（Go / eino） |
|---|---|
| `contextvars.ContextVar` 传 user_id | `context.Context` + `context.WithValue`（类型安全 key，`internal/sandbox/context.go`） |
| `@before_tool_call` 可**改写 / 阻断**入参 | 装饰器实现 `tool.InvokableTool`，委托前改写 / 阻断（`internal/sandbox/secured_tool.go`） |
| `@after_tool_call` / 观测类 Hook | `callbacks.Handler`，`OnStart` / `OnEnd` 只读观测（`internal/sandbox/audit.go`） |
| `Path.resolve()` + `relative_to()` | `filepath.Abs` + `filepath.Join`（自带 `Clean`）+ `filepath.Rel`（`internal/sandbox/safepath.go`） |
| `FileWriterTool` / `FileReadTool` | `utils.InferTool` 自造 `write_file` / `read_file`（`internal/todo/tools.go`） |
| `crew.kickoff()` 多轮 | `adk.NewChatModelAgent` + `adk.NewRunner(...).Query(...)`（`internal/todo/agent.go`） |

## 为什么拦截用「装饰器」而不是 callbacks

这是本课程最容易踩错的点：

- **callbacks 只能观测，不能改写 / 阻断。** eino 的 `callbacks.Handler` 在工具执行前后被调用，拿到的是入参/结果的副本，返回值是 `context.Context`，没有任何途径去修改真正传给工具的参数，也无法阻止工具执行。它天然对应课程里「观测类」的 `@after_tool_call`。
- **真正的拦截必须实现 `tool.InvokableTool`。** 要在工具执行前改写路径、或在发现穿越时直接短路返回，只能在工具调用链路**内部**动手。于是我们用装饰器模式包一层：`securedFileTool` 持有内部真实工具 `inner`，`Info` 透传（模型对这层无感），`InvokableRun` 在委托 `inner` 之前完成「取 user_id → 拼工作空间 → 校验/重写路径 → 阻断穿越」。这才对应课程里能改写/阻断的 `@before_tool_call`。

两者配合 = 课程的 before（装饰器）+ after（callbacks）两类 Hook。

**铁律**：装饰器的所有阻断 / 错误分支都返回 `(说明字符串, nil)`，**绝不返回 Go error**。因为错误信息会作为工具结果回传给模型，让模型据此调整（例如换个合法文件名），而不是让整条链路崩掉。

## 路径穿越两阶段防御

`sandbox.SafePath(baseDir, userInput)` 分两步把用户输入限制在 `baseDir`（即 `./workspace/{uid}`）之内：

1. **阶段一**：把 `userInput` 经 `filepath.Abs` 取绝对路径，若它已经落在 `baseDir` 内（`filepath.Rel` 结果不以 `..` 开头、也不是绝对路径），直接放行。这覆盖「传入的本就是工作空间内绝对路径」的情况。
2. **阶段二**：用 `filepath.Join(baseDir, userInput)`，`Join` 自带 `Clean` 会把 `../` 做词法归一；再 `filepath.Rel` 校验结果是否仍在 `baseDir` 内，越界就返回 `ErrPathTraversal`。

**与 Python 的关键差异**：Go 的 `filepath.Join` / `filepath.Clean` 是**纯词法**归一，**不跟随符号链接（symlink）**；而 Python 的 `Path.resolve()` 会真实解析软链。对于本课程防御的 `../` 穿越，词法归一已经足够。若要防「工作空间内有一个指向外部的软链」这种更强攻击，可用 `filepath.EvalSymlinks` 做加固（代码里以注释形式说明，不作主线）。

## 目录结构

```
internal/sandbox/        # 安全核心（与业务无关，可复用）
  context.go             # 请求级 user_id：WithUserID / UserID
  safepath.go            # 两阶段路径防御：SafePath / ErrPathTraversal
  secured_tool.go        # 工具装饰器（before_tool_call）：NewSecuredFileTool
  audit.go               # 观测型审计 Hook（after_tool_call）：NewAuditHandler
  safepath_test.go
  secured_tool_test.go
internal/todo/           # 业务 demo
  tools.go               # write_file / read_file（字段统一 "path"）并装饰
  agent.go               # 待办清单管理助手 Agent + Service.Run
  README.md              # 本讲义
cmd/todo/
  main.go                # CLI：固定 uid 跑四轮正常对话 + 第 5 轮模拟注入拦截
workspace/{uid}/TODO.md  # 运行期产物（不手写、不提交）
```

## 如何跑

**无需 API Key（主验证手段）** —— 安全核心的单元测试覆盖了路径穿越与隔离保证：

```bash
cd /Users/cms/work/mas_demo_m2l7
go test ./internal/sandbox/...
```

**需要 API Key（端到端演示，人工）**：

```bash
cd /Users/cms/work/mas_demo_m2l7
export QWEN_API_KEY=...   # 或 DASHSCOPE_API_KEY
go run ./cmd/todo
```

预期：四轮正常对话各有自然语言回复；`./workspace/1234567890/TODO.md` 被创建并写入待办行（第 4 轮把「买牛奶」也标记完成后，「买牛奶」「写周报」都已打勾）。第 5 轮模拟注入攻击，直接以工具入参注入 `../../../../etc/passwd` 被安全装饰器拦截，打印 `拦截返回： 非法的路径：../../../../etc/passwd（路径超出工作空间范围）`，随后用合法入参读回 `TODO.md`，证明工作空间内的真实清单未被注入污染。

## 学习检查清单

- [ ] 能说清为什么「拦截/改写」必须用装饰器（实现 `tool.InvokableTool`），而 callbacks 只能观测。
- [ ] 能解释 user_id 为什么走 `context.Context` 而不是全局变量（请求级、并发安全，对应 `contextvars`）。
- [ ] 能复述两阶段路径防御的流程，以及 Go 词法 `Clean` 与 Python `resolve()` 在 symlink 上的差异。
- [ ] 理解为什么装饰器阻断时返回 `(字符串, nil)` 而不是 error（错误信息要回传给模型）。
- [ ] 理解跨轮状态靠 `TODO.md` 文件持久化，而非对话历史——每轮 `Query` 都是独立的。
- [ ] 能指出路径字段为何统一成 `path`（单一语义，装饰器拦截无需区分字段名）。
