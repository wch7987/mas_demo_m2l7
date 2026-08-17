package llm

import "fmt"

var endpoints = map[string]string{
	"cn":      "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions",
	"intl":    "https://dashscope-intl.aliyuncs.com/compatible-mode/v1/chat/completions",
	"finance": "https://dashscope-finance.aliyuncs.com/compatible-mode/v1/chat/completions",
}

// ResolveEndpoint returns the chat completions URL for the given region or base URL.
func ResolveEndpoint(region, baseURL string) (string, error) {
	if baseURL != "" {
		baseURL = trimRightSlash(baseURL)
		if hasSuffix(baseURL, "/chat/completions") {
			return baseURL, nil
		}
		return baseURL + "/chat/completions", nil
	}

	endpoint, ok := endpoints[region]
	if !ok {
		return "", fmt.Errorf("不支持的地域: %s，支持: cn, intl, finance", region)
	}
	return endpoint, nil
}

func trimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
