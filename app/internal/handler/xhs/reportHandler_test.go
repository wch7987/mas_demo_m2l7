package xhs

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cms/mas_demo_m2l7/app/internal/config"
	"github.com/cms/mas_demo_m2l7/app/internal/svc"
	imageutil "github.com/cms/mas_demo_m2l7/internal/image"
	xhspkg "github.com/cms/mas_demo_m2l7/internal/xhs"
)

type flushRecorder struct {
	*httptest.ResponseRecorder
}

func (f *flushRecorder) Flush() {}

func TestReportHandlerSuccessSSE(t *testing.T) {
	root := t.TempDir()
	svcCtx := &svc.ServiceContext{
		Config: config.Config{},
		Service: &xhspkg.Service{
			Store: &imageutil.Store{OutputDir: root, MaxImages: 20, MaxSize: 1024, Quality: 85},
			Runner: xhspkg.StubRunner{
				Report: "原始创作意图: 地中海饮食\n生成笔记标题: 测试标题\n生成笔记正文: 测试正文",
			},
		},
	}

	body, ctype := multipartJPEG(t, "我想分享最近开始用地中海饮食减脂")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/xhs/notes/report", body)
	req.Header.Set("Content-Type", ctype)
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}

	ReportHandler(svcCtx).ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("content-type=%s", ct)
	}
	text := rec.Body.String()
	if !strings.Contains(text, `"type":"start"`) {
		t.Fatalf("missing start: %s", text)
	}
	if !strings.Contains(text, `"type":"done"`) {
		t.Fatalf("missing done: %s", text)
	}
	if !strings.Contains(text, "测试标题") {
		t.Fatalf("missing report: %s", text)
	}
}

func TestReportHandlerNoImagesSSEError(t *testing.T) {
	root := t.TempDir()
	svcCtx := &svc.ServiceContext{
		Service: &xhspkg.Service{
			Store:  &imageutil.Store{OutputDir: root, MaxImages: 20},
			Runner: xhspkg.StubRunner{Report: "should not run"},
		},
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("idea_text", "测试")
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/xhs/notes/report", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	ReportHandler(svcCtx).ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status=%d", rec.Code)
	}
	text := rec.Body.String()
	if !strings.Contains(text, `"type":"error"`) {
		t.Fatalf("expected error event: %s", text)
	}
}

func multipartJPEG(t *testing.T, idea string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("idea_text", idea); err != nil {
		t.Fatal(err)
	}
	part, err := w.CreateFormFile("images", "cup.jpg")
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 12, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 12; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 80, B: 20, A: 255})
		}
	}
	if err := jpeg.Encode(part, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, w.FormDataContentType()
}
