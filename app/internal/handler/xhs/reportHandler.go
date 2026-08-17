package xhs

import (
	"io"
	"net/http"
	"path/filepath"

	"github.com/cms/mas_demo_m2l7/app/internal/logic/xhs"
	"github.com/cms/mas_demo_m2l7/app/internal/sse"
	"github.com/cms/mas_demo_m2l7/app/internal/svc"
	imageutil "github.com/cms/mas_demo_m2l7/internal/image"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const maxMultipartMemory = 64 << 20 // 64MB

func ReportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		idea := r.FormValue("idea_text")
		files, err := readImages(r)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			httpx.ErrorCtx(r.Context(), w, http.ErrNotSupported)
			return
		}

		sse.InitHeaders(w)
		flusher.Flush()

		l := xhs.NewReportLogic(r.Context(), svcCtx)
		err = l.Run(idea, files, func(event sse.Event) error {
			return sse.Write(w, flusher, event)
		})
		if err != nil {
			_ = sse.Write(w, flusher, sse.Event{
				Type:    "error",
				Message: err.Error(),
			})
		}
	}
}

func readImages(r *http.Request) ([]imageutil.NamedFile, error) {
	if r.MultipartForm == nil {
		return nil, nil
	}
	headers := r.MultipartForm.File["images"]
	out := make([]imageutil.NamedFile, 0, len(headers))
	for _, fh := range headers {
		f, err := fh.Open()
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			return nil, err
		}
		name := fh.Filename
		if name == "" {
			name = filepath.Base("image.jpg")
		}
		out = append(out, imageutil.NamedFile{Name: name, Content: content})
	}
	return out, nil
}
