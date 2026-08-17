package imageutil

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	got := SanitizeFilename("photo.jpg", "fallback.jpg", 255)
	if got != "photo.jpg" {
		t.Fatalf("got %s", got)
	}
	got = SanitizeFilename("", "fallback.jpg", 255)
	if got != "fallback.jpg" {
		t.Fatalf("got %s", got)
	}
	got = SanitizeFilename("../a/b.jpg", "fallback.jpg", 255)
	if got != ".._a_b.jpg" {
		t.Fatalf("got %s", got)
	}
	got = SanitizeFilename("a<b>.jpg", "fallback.jpg", 255)
	if got != "a_b_.jpg" {
		t.Fatalf("got %s", got)
	}
	long := stringsRepeat("x", 300) + ".jpg"
	got = SanitizeFilename(long, "fallback.jpg", 255)
	if len(got) > 255 {
		t.Fatalf("len=%d", len(got))
	}
	if filepath.Ext(got) != ".jpg" {
		t.Fatalf("ext=%s", filepath.Ext(got))
	}
}

func TestValidateCount(t *testing.T) {
	s := &Store{MaxImages: 2}
	if err := s.ValidateCount(0); err == nil {
		t.Fatal("expected error")
	}
	if err := s.ValidateCount(3); err == nil {
		t.Fatal("expected error")
	}
	if err := s.ValidateCount(1); err != nil {
		t.Fatal(err)
	}
}

func TestCompressFileLongEdge(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "big.jpg")
	if err := writeJPEG(src, 2000, 800); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "out.jpg")
	out, err := CompressFile(src, dst, 1024, 85)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if max(b.Dx(), b.Dy()) > 1024 {
		t.Fatalf("size %dx%d", b.Dx(), b.Dy())
	}
}

func TestSaveAllAndCleanup(t *testing.T) {
	dir := t.TempDir()
	s := &Store{OutputDir: dir, MaxSize: 1024, Quality: 85, MaxImages: 20}
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 10, B: 10, A: 255})
		}
	}
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	outDir, images, err := s.SaveAll("abc123", []NamedFile{{Name: "cup.jpg", Content: buf.Bytes()}})
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 || images[0].ImageID != "img_0" {
		t.Fatalf("%+v", images)
	}
	if _, err := os.Stat(images[0].LocalPath); err != nil {
		t.Fatal(err)
	}
	Cleanup(outDir)
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Fatalf("dir should be gone: %v", err)
	}
}

func TestFileToDataURL(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.jpg")
	if err := writeJPEG(src, 8, 8); err != nil {
		t.Fatal(err)
	}
	url, err := FileToDataURL(src)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(url), []byte("data:image/jpeg;base64,")) {
		t.Fatalf("unexpected prefix: %s", url[:min(40, len(url))])
	}
}

func writeJPEG(path string, w, h int) error {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 30, G: 80, B: 40, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 80})
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
