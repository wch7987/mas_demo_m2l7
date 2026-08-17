package xhs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	imageutil "github.com/cms/mas_demo_m2l7/internal/image"
)

// Service saves images, runs the flow, and always cleans the temp directory.
type Service struct {
	Store  *imageutil.Store
	Runner Runner
}

func (s *Service) Generate(ctx context.Context, idea string, files []imageutil.NamedFile, onProgress ProgressFunc) (string, error) {
	if s == nil || s.Store == nil || s.Runner == nil {
		return "", fmt.Errorf("service not initialized")
	}
	if strings.TrimSpace(idea) == "" {
		return "", fmt.Errorf("idea_text 不能为空")
	}
	if err := s.Store.ValidateCount(len(files)); err != nil {
		return "", err
	}

	runID := newRunID()
	dir, saved, err := s.Store.SaveAll(runID, files)
	defer imageutil.Cleanup(dir)
	if err != nil {
		return "", err
	}

	images := make([]ImageInput, 0, len(saved))
	for _, img := range saved {
		images = append(images, ImageInput{
			ImageID:   img.ImageID,
			FileName:  img.FileName,
			LocalPath: img.LocalPath,
		})
	}

	report, err := s.Runner.Run(ctx, idea, images, onProgress)
	if err != nil {
		return "", err
	}
	if report == "" {
		return "", fmt.Errorf("生成报告为空")
	}
	return report, nil
}

func newRunID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "00000000"
	}
	return hex.EncodeToString(b)
}
