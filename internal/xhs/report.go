package xhs

import (
	"fmt"
	"strings"
)

func BuildReport(idea string, edit *ImageEditBatchReport, seo *SEOOptimizedNote) string {
	var b strings.Builder
	b.WriteString("原始创作意图: ")
	b.WriteString(idea)
	b.WriteString("\n生成笔记标题: ")
	b.WriteString(seo.OptimizedTitle)
	b.WriteString("\n生成笔记正文: ")
	b.WriteString(seo.OptimizedContent)
	b.WriteString("\n生成笔记图片顺序: ")
	b.WriteString(fmt.Sprintf("%v", seo.OptimizedPictureOrder))
	b.WriteString("\n生成笔记标签: ")
	b.WriteString(fmt.Sprintf("%v", seo.Tags))
	b.WriteString("\n生成笔记图片编辑方案: \n")
	for _, img := range edit.ImagesEditPlan {
		b.WriteString("图片ID: ")
		b.WriteString(img.ImageID)
		b.WriteString("\n图片编辑方案: ")
		b.WriteString(img.OverallEditStrategy)
		b.WriteString("\n图片剪裁建议: ")
		b.WriteString(img.CropSuggestion)
		b.WriteString("\n图片亮度/对比度/饱和度调整建议: ")
		b.WriteString(img.LightColorAdjustment)
		b.WriteString("\n图片滤镜建议: ")
		b.WriteString(img.FilterSuggestion)
		b.WriteString("\n图片文字建议: ")
		b.WriteString(img.TextOverlaySuggestion)
		b.WriteString("\n图片美颜建议: ")
		b.WriteString(img.BeautyAdjustmentSuggestion)
		b.WriteString("\n图片是否建议作为首图: ")
		b.WriteString(fmt.Sprintf("%v", img.IsRecommendedAsCover))
		b.WriteString("\n图片需要规避的审美风险/平台审核风险: ")
		b.WriteString(img.RiskAndPitfallNotes)
		b.WriteString("\n")
	}
	return b.String()
}
