package xhs

import "testing"

func TestAlignVisualBatchByIndex(t *testing.T) {
	images := []ImageInput{
		{ImageID: "img_0", FileName: "a.jpg"},
		{ImageID: "img_1", FileName: "b.jpg"},
	}
	got, err := alignVisualBatch("卖杯子", images, &visualBatchSubmit{
		ImagesVisual: VisualList{
			{SubjectDescription: "绿杯", AtmosphereVibe: "静", HighlightFeature: "裂纹"},
			{SubjectDescription: "特写", AtmosphereVibe: "暖", HighlightFeature: "反光"},
		},
		Summary: "复古静谧",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.UserRawIntent != "卖杯子" || got.Summary != "复古静谧" {
		t.Fatalf("%+v", got)
	}
	if got.ImagesVisual[0].ImageID != "img_0" || got.ImagesVisual[0].FileName != "a.jpg" {
		t.Fatalf("%+v", got.ImagesVisual[0])
	}
	if got.ImagesVisual[1].ImageID != "img_1" {
		t.Fatalf("%+v", got.ImagesVisual[1])
	}
}

func TestAlignVisualBatchCountMismatch(t *testing.T) {
	_, err := alignVisualBatch("x", []ImageInput{{ImageID: "img_0"}}, &visualBatchSubmit{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAlignEditBatchByID(t *testing.T) {
	images := []ImageInput{
		{ImageID: "img_0", FileName: "a.jpg"},
		{ImageID: "img_1", FileName: "b.jpg"},
	}
	got, err := alignEditBatch(images, &editBatchSubmit{
		ImagesEditPlan: EditList{
			{ImageID: "img_1", OverallEditStrategy: "提亮第二张"},
			{ImageID: "img_0", OverallEditStrategy: "裁第一张"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ImagesEditPlan[0].OverallEditStrategy != "裁第一张" || got.ImagesEditPlan[0].FileName != "a.jpg" {
		t.Fatalf("%+v", got.ImagesEditPlan[0])
	}
	if got.ImagesEditPlan[1].OverallEditStrategy != "提亮第二张" {
		t.Fatalf("%+v", got.ImagesEditPlan[1])
	}
}
