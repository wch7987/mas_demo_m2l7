package xhs

import (
	"context"

	"github.com/cms/mas_demo_m2l7/app/internal/sse"
	"github.com/cms/mas_demo_m2l7/app/internal/svc"
	imageutil "github.com/cms/mas_demo_m2l7/internal/image"
	xhspkg "github.com/cms/mas_demo_m2l7/internal/xhs"
	"github.com/zeromicro/go-zero/core/logx"
)

type ReportLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReportLogic {
	return &ReportLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReportLogic) Run(idea string, files []imageutil.NamedFile, write func(sse.Event) error) error {
	report, err := l.svcCtx.Service.Generate(l.ctx, idea, files, func(ev xhspkg.ProgressEvent) {
		_ = write(sse.Event{
			Type:    ev.Type,
			Task:    ev.Task,
			Agent:   ev.Agent,
			Message: ev.Message,
			Content: ev.Content,
			Step:    ev.Step,
		})
	})
	if err != nil {
		return err
	}
	return write(sse.Event{
		Type:    "done",
		Content: report,
	})
}
