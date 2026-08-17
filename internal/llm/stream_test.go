package llm

import (
	"strings"
	"testing"
)

func TestReadSSEStream(t *testing.T) {
	body := strings.NewReader(
		"data: {\"choices\":[{\"delta\":{\"content\":\"\\u4f60\"}}]}\n\n" +
			"data: {\"choices\":[{\"delta\":{\"content\":\"\\u597d\"}}]}\n\n" +
			"data: [DONE]\n\n",
	)
	var chunks []string
	err := readSSEStream(body, func(delta string) error {
		chunks = append(chunks, delta)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 || chunks[0] != "\u4f60" || chunks[1] != "\u597d" {
		t.Fatalf("unexpected chunks: %#v", chunks)
	}
}
