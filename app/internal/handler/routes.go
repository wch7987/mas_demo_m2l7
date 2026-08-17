package handler

import (
	"net/http"

	xhsnote "github.com/cms/mas_demo_m2l7/app/internal/handler/xhs"
	"github.com/cms/mas_demo_m2l7/app/internal/svc"
	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/api/v1/xhs/notes/report",
				Handler: xhsnote.ReportHandler(serverCtx),
			},
		},
	)
}
