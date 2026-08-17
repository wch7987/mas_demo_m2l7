package imageutil

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var unsafeNameChars = regexp.MustCompile(`[<>:"|?*]`)

// SanitizeFilename strips path separators and unsafe characters.
func SanitizeFilename(filename, fallback string, maxLength int) string {
	if maxLength <= 0 {
		maxLength = 255
	}
	name := filename
	if strings.TrimSpace(name) == "" {
		name = fallback
	}
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = unsafeNameChars.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	if len(name) <= maxLength {
		return name
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	remain := maxLength - len(ext)
	if remain <= 0 {
		if len(name) > maxLength {
			return name[:maxLength]
		}
		return name
	}
	if len(stem) > remain {
		stem = stem[:remain]
	}
	return stem + ext
}

// NamedFile is an in-memory uploaded or CLI-copied image.
type NamedFile struct {
	Name    string
	Content []byte
}

// PrepareResult is the temp directory and parsed image inputs after save/compress.
type PrepareResult struct {
	RunID string
	Dir   string
}

// Store saves, compresses, and later cleans request-scoped image directories.
type Store struct {
	OutputDir string
	MaxSize   int
	Quality   int
	MaxImages int
}

func (s *Store) maxImages() int {
	if s.MaxImages <= 0 {
		return DefaultMaxCount
	}
	return s.MaxImages
}

func (s *Store) maxSize() int {
	if s.MaxSize == 0 {
		return DefaultMaxSize
	}
	return s.MaxSize
}

func (s *Store) quality() int {
	if s.Quality <= 0 {
		return DefaultQuality
	}
	return s.Quality
}

func (s *Store) outputRoot() string {
	if s.OutputDir == "" {
		return "data/output"
	}
	return s.OutputDir
}

// ValidateCount checks image count before saving.
func (s *Store) ValidateCount(n int) error {
	if n <= 0 {
		return fmt.Errorf("至少需要上传一张图片")
	}
	if n > s.maxImages() {
		return fmt.Errorf("最多支持上传 %d 张图片", s.maxImages())
	}
	return nil
}

// SaveAll writes files under data/output/xhs_note/<runID>/, compresses, and returns local paths.
func (s *Store) SaveAll(runID string, files []NamedFile) (dir string, images []SavedImage, err error) {
	if err := s.ValidateCount(len(files)); err != nil {
		return "", nil, err
	}
	dir = filepath.Join(s.outputRoot(), "xhs_note", runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, fmt.Errorf("创建临时目录失败: %w", err)
	}

	out := make([]SavedImage, 0, len(files))
	for i, f := range files {
		imageID := fmt.Sprintf("img_%d", i)
		fallback := imageID + ".jpg"
		safeName := SanitizeFilename(f.Name, fallback, 255)
		target := filepath.Join(dir, safeName)
		if err := os.WriteFile(target, f.Content, 0o644); err != nil {
			return dir, nil, fmt.Errorf("保存图片失败: %w", err)
		}
		local := target
		compressed, cErr := CompressFile(target, target, s.maxSize(), s.quality())
		if cErr == nil && compressed != "" {
			local = compressed
			if compressed != target {
				_ = os.Remove(target)
			}
			safeName = filepath.Base(local)
		}
		out = append(out, SavedImage{
			ImageID:   imageID,
			FileName:  safeName,
			LocalPath: local,
		})
	}
	return dir, out, nil
}

// SavedImage is the on-disk representation after upload/copy.
type SavedImage struct {
	ImageID   string
	FileName  string
	LocalPath string
}

// Cleanup removes the request temp directory.
func Cleanup(dir string) {
	if dir == "" {
		return
	}
	_ = os.RemoveAll(dir)
}
