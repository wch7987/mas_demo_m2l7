package sandbox

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafePath(t *testing.T) {
	base := t.TempDir()

	tests := []struct {
		name       string
		input      string
		wantErr    error  // 非 nil 时用 errors.Is 校验
		wantSuffix string // wantErr 为 nil 时校验返回路径后缀（ToSlash 归一）
	}{
		{
			name:       "裸文件名落在 base 内",
			input:      "CRONTAB.md",
			wantSuffix: "CRONTAB.md",
		},
		{
			name:       "base 内绝对路径直通",
			input:      filepath.Join(base, "a.md"),
			wantSuffix: "a.md",
		},
		{
			name:    "深度穿越被拦截",
			input:   "../../etc/passwd",
			wantErr: ErrPathTraversal,
		},
		{
			name:       "归一后仍在 base 内",
			input:      "a/../b.md",
			wantSuffix: "b.md",
		},
		{
			name:    "浅穿越被拦截",
			input:   "../x",
			wantErr: ErrPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafePath(base, tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("期望错误 %v，实际 err=%v got=%q", tt.wantErr, err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("不期望错误，却得到 err=%v", err)
			}
			// 返回必须是落在 base 内的路径，且后缀匹配。
			if rel, relErr := filepath.Rel(base, got); relErr != nil || !isInside(rel) {
				t.Fatalf("返回路径 %q 不在 base %q 内（rel=%q err=%v）", got, base, rel, relErr)
			}
			if !strings.HasSuffix(filepath.ToSlash(got), tt.wantSuffix) {
				t.Fatalf("返回路径 %q 不以 %q 结尾", got, tt.wantSuffix)
			}
		})
	}
}
