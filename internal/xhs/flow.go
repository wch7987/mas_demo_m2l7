package xhs

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/zeromicro/go-zero/core/logx"
)

// FlowConfig builds a per-request three-phase XHS note pipeline.
type FlowConfig struct {
	TextModel  model.ChatModel
	ImageModel model.ChatModel
	MaxIter    int
}

// Flow is created per request; do not reuse Agent instances across requests.
type Flow struct {
	cfg FlowConfig
}

func NewFlow(cfg FlowConfig) (*Flow, error) {
	if cfg.TextModel == nil {
		return nil, fmt.Errorf("TextModel is required")
	}
	if cfg.ImageModel == nil {
		cfg.ImageModel = cfg.TextModel
	}
	cfg.MaxIter = clampMaxIter(cfg.MaxIter)
	return &Flow{cfg: cfg}, nil
}

func (f *Flow) Run(ctx context.Context, idea string, images []ImageInput, onProgress ProgressFunc) (string, error) {
	if f == nil {
		return "", fmt.Errorf("flow is nil")
	}
	if len(images) == 0 {
		return "", fmt.Errorf("本次请求未上传任何图片")
	}

	emit(onProgress, ProgressEvent{
		Type:    "start",
		Message: fmt.Sprintf("xhs note started, images=%d", len(images)),
	})

	logx.Infof("xhs_note_visual_phase_start image_count=%d", len(images))
	visual, err := runVisualPhase(ctx, f.cfg.ImageModel, f.cfg.MaxIter, idea, images, onProgress)
	if err != nil {
		return "", err
	}

	logx.Infof("xhs_note_edit_phase_start image_count=%d", len(images))
	edit, err := runEditPhase(ctx, f.cfg.ImageModel, f.cfg.MaxIter, idea, images, visual, onProgress)
	if err != nil {
		return "", err
	}

	logx.Info("xhs_note_content_phase_start")
	content, err := runContentPhase(ctx, f.cfg.TextModel, f.cfg.MaxIter, idea, visual, edit, onProgress)
	if err != nil {
		return "", err
	}

	report := BuildReport(idea, edit, content.SEO)
	logx.Info("xhs_note_flow_success")
	return report, nil
}

// Runner is the HTTP/CLI entry that can be swapped in tests.
type Runner interface {
	Run(ctx context.Context, idea string, images []ImageInput, onProgress ProgressFunc) (string, error)
}

type liveRunner struct {
	newFlow func() (*Flow, error)
}

func NewLiveRunner(textModel, imageModel model.ChatModel, maxIter int) Runner {
	return &liveRunner{
		newFlow: func() (*Flow, error) {
			return NewFlow(FlowConfig{
				TextModel:  textModel,
				ImageModel: imageModel,
				MaxIter:    maxIter,
			})
		},
	}
}

func (r *liveRunner) Run(ctx context.Context, idea string, images []ImageInput, onProgress ProgressFunc) (string, error) {
	flow, err := r.newFlow()
	if err != nil {
		return "", err
	}
	return flow.Run(ctx, idea, images, onProgress)
}

// StubRunner is used by HTTP contract tests.
type StubRunner struct {
	Report string
	Err    error
}

func (s StubRunner) Run(_ context.Context, _ string, images []ImageInput, onProgress ProgressFunc) (string, error) {
	emit(onProgress, ProgressEvent{Type: "start", Message: "stub start"})
	if s.Err != nil {
		return "", s.Err
	}
	for i, img := range images {
		emit(onProgress, ProgressEvent{
			Type:    "task_done",
			Task:    "visual:" + img.ImageID,
			Agent:   visualAnalystName,
			Message: "stub visual " + img.FileName,
			Step:    i + 1,
		})
	}
	emit(onProgress, ProgressEvent{Type: "task_done", Task: "content", Agent: seoName, Message: "stub content"})
	report := s.Report
	if report == "" {
		report = "stub report"
	}
	return report, nil
}
