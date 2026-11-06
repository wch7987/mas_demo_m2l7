package sandbox

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// fakeTool 是内存假工具，实现 tool.InvokableTool，用于断言装饰器是否放行以及重写后的入参。
type fakeTool struct {
	called   bool
	lastArgs string
}

func (f *fakeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "write_file"}, nil
}

func (f *fakeTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	f.called = true
	f.lastArgs = argumentsInJSON
	return "ok", nil
}

// pathFromArgs 从工具入参 JSON 中取出 path 字段。
func pathFromArgs(t *testing.T, argsJSON string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &m); err != nil {
		t.Fatalf("解析 lastArgs 失败：%v", err)
	}
	path, _ := m["path"].(string)
	return path
}

// chdirTemp 把 CWD 切到临时目录，避免装饰器的 ./workspace 产物污染仓库。
// go 1.23.7 没有 t.Chdir（Go 1.24 才有），故用 os.Chdir + t.Cleanup 还原。
func chdirTemp(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前目录失败：%v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("切换到临时目录失败：%v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestSecuredFileToolRewritesPath(t *testing.T) {
	chdirTemp(t)

	const uid = "1234567890"
	fake := &fakeTool{}
	secured := NewSecuredFileTool(fake, "path")
	ctx := WithUserID(context.Background(), uid)

	out, err := secured.InvokableRun(ctx, `{"path":"CRONTAB.md","content":"x"}`)
	if err != nil {
		t.Fatalf("不期望 Go error，却得到 %v", err)
	}
	if out != "ok" {
		t.Fatalf("期望放行返回内部工具结果 ok，实际 %q", out)
	}
	if !fake.called {
		t.Fatal("期望内部工具被调用")
	}
	path := pathFromArgs(t, fake.lastArgs)
	if !strings.HasSuffix(filepath.ToSlash(path), "workspace/"+uid+"/CRONTAB.md") {
		t.Fatalf("重写后的路径 %q 未落在 workspace/%s/ 下", path, uid)
	}
}

func TestSecuredFileToolMissingUID(t *testing.T) {
	chdirTemp(t)

	fake := &fakeTool{}
	secured := NewSecuredFileTool(fake, "path")

	out, err := secured.InvokableRun(context.Background(), `{"path":"CRONTAB.md"}`)
	if err != nil {
		t.Fatalf("不期望 Go error，却得到 %v", err)
	}
	if !strings.Contains(out, "缺少必要的上下文信息") {
		t.Fatalf("期望返回缺少上下文的提示，实际 %q", out)
	}
	if fake.called {
		t.Fatal("缺少 uid 时不应调用内部工具")
	}
}

func TestSecuredFileToolBlocksTraversal(t *testing.T) {
	chdirTemp(t)

	fake := &fakeTool{}
	secured := NewSecuredFileTool(fake, "path")
	ctx := WithUserID(context.Background(), "1234567890")

	out, err := secured.InvokableRun(ctx, `{"path":"../../etc/passwd","content":"x"}`)
	if err != nil {
		t.Fatalf("不期望 Go error，却得到 %v", err)
	}
	if !strings.Contains(out, "非法的路径") {
		t.Fatalf("期望返回非法路径提示，实际 %q", out)
	}
	if fake.called {
		t.Fatal("穿越路径时不应调用内部工具")
	}
}

func TestSecuredFileToolTenantIsolation(t *testing.T) {
	chdirTemp(t)

	fakeA := &fakeTool{}
	fakeB := &fakeTool{}
	securedA := NewSecuredFileTool(fakeA, "path")
	securedB := NewSecuredFileTool(fakeB, "path")

	const uidA = "aaa111"
	const uidB = "bbb222"
	if _, err := securedA.InvokableRun(WithUserID(context.Background(), uidA), `{"path":"CRONTAB.md"}`); err != nil {
		t.Fatalf("租户 A 调用出错：%v", err)
	}
	if _, err := securedB.InvokableRun(WithUserID(context.Background(), uidB), `{"path":"CRONTAB.md"}`); err != nil {
		t.Fatalf("租户 B 调用出错：%v", err)
	}

	pathA := filepath.ToSlash(pathFromArgs(t, fakeA.lastArgs))
	pathB := filepath.ToSlash(pathFromArgs(t, fakeB.lastArgs))
	if !strings.Contains(pathA, "workspace/"+uidA+"/") {
		t.Fatalf("租户 A 路径 %q 未隔离到自己目录", pathA)
	}
	if !strings.Contains(pathB, "workspace/"+uidB+"/") {
		t.Fatalf("租户 B 路径 %q 未隔离到自己目录", pathB)
	}
	if pathA == pathB {
		t.Fatalf("两个租户的路径不应相同：%q", pathA)
	}
}
