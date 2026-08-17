package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	imageutil "github.com/cms/mas_demo_m2l7/internal/image"
	"github.com/cms/mas_demo_m2l7/internal/llm"
	"github.com/cms/mas_demo_m2l7/internal/xhs"
)

func main() {
	idea := flag.String("idea", "", "笔记创作意图")
	imagesFlag := flag.String("images", "", "本地图片路径，逗号分隔")
	flag.Parse()

	if strings.TrimSpace(*idea) == "" || strings.TrimSpace(*imagesFlag) == "" {
		fmt.Fprintf(os.Stderr, "用法: go run ./cmd/xhs -idea \"想卖马克杯\" -images a.jpg,b.jpg\n")
		os.Exit(2)
	}

	paths := splitImages(*imagesFlag)
	files := make([]imageutil.NamedFile, 0, len(paths))
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "读取图片失败 %s: %v\n", p, err)
			os.Exit(1)
		}
		files = append(files, imageutil.NamedFile{Name: filepath.Base(p), Content: raw})
	}

	llmCfg, err := llm.ConfigFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置错误: %v\n", err)
		os.Exit(1)
	}
	if os.Getenv("QWEN_IMAGE_MODEL") == "" {
		llmCfg.ImageModel = "qwen3-vl-plus"
	}

	chatModel, err := llm.NewEinoChatModel(llmCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化 LLM 失败: %v\n", err)
		os.Exit(1)
	}
	imageModel := llm.WithModel(chatModel, llmCfg.ImageModel)

	svc := &xhs.Service{
		Store: &imageutil.Store{
			OutputDir: "data/output",
			MaxSize:   imageutil.DefaultMaxSize,
			Quality:   imageutil.DefaultQuality,
			MaxImages: imageutil.DefaultMaxCount,
		},
		Runner: xhs.NewLiveRunner(chatModel, imageModel, 3),
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("m2l7 小红书笔记：视觉 → 修图 → 策略/文案/SEO")
	fmt.Println(strings.Repeat("=", 80))

	report, err := svc.Generate(context.Background(), *idea, files, func(ev xhs.ProgressEvent) {
		if ev.Message != "" {
			fmt.Printf("[%s] %s\n", ev.Type, ev.Message)
		}
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "执行失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("最终报告")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println(report)
}

func splitImages(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
