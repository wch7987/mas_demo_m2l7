package xhs

// ProgressEvent is mapped to SSE frames by the HTTP handler.
type ProgressEvent struct {
	Type    string
	Task    string
	Agent   string
	Message string
	Content string
	Step    int
}

// ProgressFunc receives pipeline progress. Nil is safe.
type ProgressFunc func(ProgressEvent)

func emit(fn ProgressFunc, ev ProgressEvent) {
	if fn != nil {
		fn(ev)
	}
}
