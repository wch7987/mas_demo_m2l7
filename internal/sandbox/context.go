// Package sandbox 实现课程「用 Hook 拦截实现安全控制与多租户隔离」的安全核心：
// 请求级 user_id 传递、路径穿越防御、工具装饰器（before_tool_call）与审计 Hook（after_tool_call）。
package sandbox

import "context"

// ctxKey 是包私有的 context key 类型，避免跨包用裸 string 作 key 造成冲突。
type ctxKey struct{ name string }

// userIDKey 用于在 context 中存取 user_id。
var userIDKey = ctxKey{"user_id"}

// WithUserID 把 user_id 绑定到 context 上。
//
// 对应 Python 版的 contextvars.ContextVar：请求级（request-scoped）、
// 天然并发安全——每条请求链路携带自己的 context，互不干扰。
func WithUserID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, userIDKey, uid)
}

// UserID 从 context 取出 user_id；第二个返回值表示是否存在且类型正确。
func UserID(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(userIDKey).(string)
	return uid, ok
}
