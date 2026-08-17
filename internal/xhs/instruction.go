package xhs

import "strings"

func instruction(role, goal, backstory, extra string) string {
	var b strings.Builder
	b.WriteString("你是")
	b.WriteString(role)
	b.WriteString("。\n目标：")
	b.WriteString(goal)
	b.WriteString("\n\n")
	b.WriteString(backstory)
	if extra != "" {
		b.WriteString("\n\n")
		b.WriteString(extra)
	}
	return b.String()
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "\n...(truncated)"
}
