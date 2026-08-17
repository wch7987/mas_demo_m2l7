package xhs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	imageutil "github.com/cms/mas_demo_m2l7/internal/image"
)

func TestContractJSONTags(t *testing.T) {
	v := ImageVisualAnalysis{ImageID: "img_0", FileName: "a.jpg", VisualDetails: []string{"x"}}
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, key := range []string{"image_id", "file_name", "visual_details"} {
		if !strings.Contains(s, key) {
			t.Fatalf("missing %s in %s", key, s)
		}
	}
}

func TestBuildReport(t *testing.T) {
	report := BuildReport("地中海饮食", &ImageEditBatchReport{
		ImagesEditPlan: []ImageEditPlan{{
			ImageID:             "img_0",
			OverallEditStrategy: "提亮",
			CropSuggestion:      "竖构图",
		}},
	}, &SEOOptimizedNote{
		OptimizedTitle:        "标题",
		OptimizedContent:      "正文",
		OptimizedPictureOrder: []string{"img_0"},
		Tags:                  []string{"饮食"},
	})
	if !strings.Contains(report, "地中海饮食") || !strings.Contains(report, "img_0") || !strings.Contains(report, "生成笔记标题") {
		t.Fatal(report)
	}
}

func TestServiceNoImages(t *testing.T) {
	svc := &Service{
		Store:  &imageutil.Store{OutputDir: t.TempDir(), MaxImages: 20},
		Runner: StubRunner{Report: "ok"},
	}
	_, err := svc.Generate(context.Background(), "idea", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "至少") {
		t.Fatalf("err=%v", err)
	}
}

func TestServiceTooManyImages(t *testing.T) {
	svc := &Service{
		Store:  &imageutil.Store{OutputDir: t.TempDir(), MaxImages: 2},
		Runner: StubRunner{Report: "ok"},
	}
	files := []imageutil.NamedFile{{Name: "a.jpg", Content: []byte("x")}, {Name: "b.jpg", Content: []byte("y")}, {Name: "c.jpg", Content: []byte("z")}}
	_, err := svc.Generate(context.Background(), "idea", files, nil)
	if err == nil || !strings.Contains(err.Error(), "最多") {
		t.Fatalf("err=%v", err)
	}
}

func TestServiceStubSuccessCleansDir(t *testing.T) {
	root := t.TempDir()
	svc := &Service{
		Store:  &imageutil.Store{OutputDir: root, MaxImages: 20, MaxSize: 1024, Quality: 85},
		Runner: StubRunner{Report: "最终报告文本"},
	}
	jpegPath := filepath.Join(t.TempDir(), "cup.jpg")
	if err := os.WriteFile(jpegPath, tinyJPEG(), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(jpegPath)
	report, err := svc.Generate(context.Background(), "地中海饮食", []imageutil.NamedFile{{Name: "cup.jpg", Content: raw}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report != "最终报告文本" {
		t.Fatal(report)
	}
	entries, _ := os.ReadDir(filepath.Join(root, "xhs_note"))
	if len(entries) != 0 {
		t.Fatalf("temp dir not cleaned: %v", entries)
	}
}

func tinyJPEG() []byte {
	// 1x1 jpeg
	return []byte{
		0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01,
		0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xff, 0xdb, 0x00, 0x43,
		0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07, 0x07, 0x07, 0x09,
		0x09, 0x08, 0x0a, 0x0c, 0x14, 0x0d, 0x0c, 0x0b, 0x0b, 0x0c, 0x19, 0x12,
		0x13, 0x0f, 0x14, 0x1d, 0x1a, 0x1f, 0x1e, 0x1d, 0x1a, 0x1c, 0x1c, 0x20,
		0x24, 0x2e, 0x27, 0x20, 0x22, 0x2c, 0x23, 0x1c, 0x1c, 0x28, 0x37, 0x29,
		0x2c, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1f, 0x27, 0x39, 0x3d, 0x38, 0x32,
		0x3c, 0x2e, 0x33, 0x34, 0x32, 0xff, 0xc0, 0x00, 0x0b, 0x08, 0x00, 0x01,
		0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xff, 0xc4, 0x00, 0x14, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x09, 0xff, 0xc4, 0x00, 0x14, 0x10, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xff, 0xda, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00, 0x3f, 0x00,
		0x7f, 0x3f, 0xff, 0xd9,
	}
}
