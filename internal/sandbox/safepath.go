package sandbox

import (
	"errors"
	"path/filepath"
	"strings"
)

// ErrPathTraversal 是路径穿越的哨兵错误（sentinel error），调用方用 errors.Is 判定。
var ErrPathTraversal = errors.New("路径超出工作空间范围")

// SafePath 把用户给的文件名/路径安全地限制在 baseDir 之内，挡住 ../ 穿越攻击。
// 对应 Python 版的 Path.resolve() + relative_to() 两步校验。
//
// 两阶段设计：
//   - 阶段一：userInput 经 filepath.Abs 归一后若已落在 base 内，直接放行
//     （覆盖「调用方已经传了 base 内绝对路径」的用例）。
//   - 阶段二：filepath.Join(base, userInput) —— Join 内部会调用 Clean 把 ../ 归一，
//     再用 filepath.Rel 校验结果是否仍在 base 内。
//
// 注意：Go 的 filepath.Join/Clean 是纯词法（lexical）归一，不会跟随 symlink 解析，
// 这点与 Python 的 Path.resolve()（会解析软链到真实路径）不同。
// 如需防御软链逃逸，可在返回前追加 filepath.EvalSymlinks 再校验一次（此处作为可选加固，
// 不作主线：本课程只需防御 ../ 词法穿越）。
func SafePath(baseDir, userInput string) (string, error) {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}

	// 阶段一：若 userInput 本身就是落在 base 内的（绝对）路径，直通返回。
	if absInput, err := filepath.Abs(userInput); err == nil {
		if rel, err := filepath.Rel(absBase, absInput); err == nil && isInside(rel) {
			return absInput, nil
		}
	}

	// 阶段二：拼接后归一再校验。
	joined := filepath.Join(absBase, userInput)
	rel, err := filepath.Rel(absBase, joined)
	if err != nil || !isInside(rel) {
		return "", ErrPathTraversal
	}
	return joined, nil
}

// isInside 判断相对路径 rel 是否仍落在 base 内部：
// 既不能等于 ".."，也不能以 "../" 开头，更不能是绝对路径。
func isInside(rel string) bool {
	return rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator)) &&
		!filepath.IsAbs(rel)
}
