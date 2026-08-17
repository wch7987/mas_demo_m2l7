package imageutil

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"

	_ "image/gif"
)

const (
	DefaultMaxSize  = 1024
	DefaultQuality  = 85
	DefaultMaxCount = 20
)

// CompressFile resizes the long edge to maxSize and re-encodes.
// JPEG (no alpha) or PNG (alpha). On failure the caller may keep the original.
func CompressFile(srcPath, dstPath string, maxSize, quality int) (string, error) {
	if maxSize < 0 {
		maxSize = 0
	}
	if quality < 1 {
		quality = DefaultQuality
	}
	if quality > 100 {
		quality = 100
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("打开图片失败: %w", err)
	}
	defer f.Close()

	img, err := decodeImage(f)
	if err != nil {
		return "", err
	}
	usePNG := hasAlpha(img)

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if maxSize > 0 && max(w, h) > maxSize {
		img = resizeLongEdge(img, maxSize)
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return "", err
	}
	outPath := dstPath
	ext := strings.ToLower(filepath.Ext(outPath))
	if usePNG {
		if ext != ".png" {
			outPath = strings.TrimSuffix(outPath, ext) + ".png"
		}
	} else if ext != ".jpg" && ext != ".jpeg" {
		outPath = strings.TrimSuffix(outPath, ext) + ".jpg"
	}

	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if usePNG {
		if err := png.Encode(out, img); err != nil {
			return "", fmt.Errorf("编码 PNG 失败: %w", err)
		}
	} else {
		if err := jpeg.Encode(out, img, &jpeg.Options{Quality: quality}); err != nil {
			return "", fmt.Errorf("编码 JPEG 失败: %w", err)
		}
	}
	return outPath, nil
}

func decodeImage(r io.Reader) (image.Image, error) {
	img, format, err := image.Decode(r)
	if err == nil {
		_ = format
		return img, nil
	}
	return nil, fmt.Errorf("无法识别图片格式: %w", err)
}

func resizeLongEdge(src image.Image, maxSize int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	var nw, nh int
	if w >= h {
		nw = maxSize
		nh = max(1, int(float64(h)*float64(maxSize)/float64(w)+0.5))
	} else {
		nh = maxSize
		nw = max(1, int(float64(w)*float64(maxSize)/float64(h)+0.5))
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

func hasAlpha(img image.Image) bool {
	switch img.(type) {
	case *image.NRGBA, *image.NRGBA64, *image.RGBA, *image.RGBA64:
		return true
	default:
		return false
	}
}

// FileToDataURL encodes a local image as a data URL for multimodal messages.
func FileToDataURL(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取图片失败: %w", err)
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("图片为空: %s", path)
	}
	mime := http.DetectContentType(raw)
	if !strings.HasPrefix(mime, "image/") {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".jpg", ".jpeg":
			mime = "image/jpeg"
		case ".png":
			mime = "image/png"
		case ".webp":
			mime = "image/webp"
		case ".gif":
			mime = "image/gif"
		default:
			mime = "image/jpeg"
		}
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

func init() {
	image.RegisterFormat("webp", "RIFF????WEBP", webp.Decode, webp.DecodeConfig)
}
