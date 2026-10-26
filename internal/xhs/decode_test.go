package xhs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/eino/components/tool/utils"
)

func TestVisualListAcceptsStringifiedArray(t *testing.T) {
	raw := []byte(`{"images_visual":"\n[{\"image_id\":\"img_0\",\"subject_description\":\"池塘\",\"atmosphere\":\"静谧\",\"visual_details\":\"水面倒影,石桥\",\"highlight_feature\":\"倒影\"}]","summary":"古城池塘"}`)
	var got visualBatchSubmit
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.ImagesVisual) != 1 {
		t.Fatalf("len=%d", len(got.ImagesVisual))
	}
	v := got.ImagesVisual[0]
	if v.SubjectDescription != "池塘" || v.AtmosphereVibe != "静谧" {
		t.Fatalf("%+v", v)
	}
	if len(v.VisualDetails) != 2 {
		t.Fatalf("details=%v", v.VisualDetails)
	}
}

func TestVisualListAcceptsRealArray(t *testing.T) {
	raw := []byte(`{"images_visual":[{"image_id":"img_0","subject_description":"池塘","atmosphere_vibe":"静","visual_details":["a","b","c"],"highlight_feature":"倒影"}],"summary":"ok"}`)
	var got visualBatchSubmit
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.ImagesVisual[0].AtmosphereVibe != "静" {
		t.Fatalf("%+v", got.ImagesVisual[0])
	}
}

func TestStringListAcceptsCSVLikeString(t *testing.T) {
	var s StringList
	if err := json.Unmarshal([]byte(`"a,b"`), &s); err != nil {
		t.Fatal(err)
	}
	if len(s) != 1 || s[0] != "a,b" {
		t.Fatalf("%v", s)
	}
}

func TestInferVisualBatchTool(t *testing.T) {
	_, err := utils.InferTool(
		"submit_visual_batch",
		"test",
		func(_ context.Context, _ *visualBatchSubmit) (string, error) {
			return "ok", nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}
