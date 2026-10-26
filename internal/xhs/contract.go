package xhs

// 契约对齐 crewai_mas_demo_m2l7/src/app/schemas/xhs_note.py。
// Eino 没有 Task.output_pydantic，用 jsonschema tag 生成 InferTool Schema。

type ImageInput struct {
	ImageID   string `json:"image_id"`
	FileName  string `json:"file_name"`
	LocalPath string `json:"local_path"`
}

type ImageVisualAnalysis struct {
	ImageID            string     `json:"image_id" jsonschema:"description=图片唯一标识，如 img_0"`
	FileName           string     `json:"file_name" jsonschema:"description=图片文件名"`
	SubjectDescription string     `json:"subject_description" jsonschema:"description=主体内容客观描述"`
	AtmosphereVibe     string     `json:"atmosphere_vibe" jsonschema:"description=情绪/氛围描述"`
	VisualDetails      StringList `json:"visual_details" jsonschema:"description=至少3个容易被忽略但重要的视觉细节"`
	ImageQualityScore  string     `json:"image_quality_score" jsonschema:"description=质量评价（1-10分+理由），综合构图、光线、清晰度"`
	HighlightFeature   string     `json:"highlight_feature" jsonschema:"description=视觉锚点：第一眼最容易被吸引的元素"`
}

type ImageEditPlan struct {
	ImageID                    string `json:"image_id" jsonschema:"description=图片唯一标识"`
	FileName                   string `json:"file_name" jsonschema:"description=图片文件名"`
	OverallEditStrategy        string `json:"overall_edit_strategy" jsonschema:"description=整体编辑思路，仅用小红书内置编辑器"`
	CropSuggestion             string `json:"crop_suggestion" jsonschema:"description=剪裁建议：横竖构图、主体位置、留白"`
	LightColorAdjustment       string `json:"light_color_adjustment" jsonschema:"description=亮度/对比度/饱和度等相对调整建议"`
	FilterSuggestion           string `json:"filter_suggestion" jsonschema:"description=小红书内置滤镜或风格描述"`
	TextOverlaySuggestion      string `json:"text_overlay_suggestion" jsonschema:"description=是否加文字、内容方向、位置，避免遮挡锚点"`
	BeautyAdjustmentSuggestion string `json:"beauty_adjustment_suggestion" jsonschema:"description=美颜建议，仅人像有效，自然不过度"`
	IsRecommendedAsCover       bool   `json:"is_recommended_as_cover" jsonschema:"description=是否建议作为首图/封面"`
	RiskAndPitfallNotes        string `json:"risk_and_pitfall_notes" jsonschema:"description=审美风险/平台审核风险"`
}

type VisualBatchReport struct {
	UserRawIntent string                `json:"user_raw_intent" jsonschema:"description=用户的原始文本意图"`
	ImagesVisual  []ImageVisualAnalysis `json:"images_visual" jsonschema:"description=所有图片的视觉分析结果"`
	Summary       string                `json:"summary" jsonschema:"description=所有图片视觉分析的一句话总结"`
}

type ImageEditBatchReport struct {
	ImagesEditPlan []ImageEditPlan `json:"images_edit_plan" jsonschema:"description=所有图片的编辑方案"`
	Summary        string          `json:"summary" jsonschema:"description=所有图片编辑方案的一句话总结"`
}

type ContentStrategyBrief struct {
	InputEvaluation       string     `json:"input_evaluation" jsonschema:"description=基于用户诉求和图片素材的综合评估，指出优势、劣势和修图建议"`
	TargetAudiencePersona string     `json:"target_audience_persona" jsonschema:"description=目标受众画像：年龄、职业、生活状态、心理诉求"`
	CorePainPoint         string     `json:"core_pain_point" jsonschema:"description=核心痛点/诉求"`
	SuggestedTitle        string     `json:"suggested_title" jsonschema:"description=建议标题：痛点场景+情绪/利益钩子+群体标签，含Emoji"`
	ContentOutline        StringList `json:"content_outline" jsonschema:"description=笔记大纲：场景引入、体验描写、干货植入、结尾引导"`
	EngagementStrategy    string     `json:"engagement_strategy" jsonschema:"description=互动策略：评论诱饵/点赞引导"`
	RetentionStrategy     string     `json:"retention_strategy" jsonschema:"description=收藏策略：给用户收藏的理由"`
	SEOKeywords           StringList `json:"seo_keywords" jsonschema:"description=3个左右必须埋入文案的长尾关键词"`
}

type CopywritingOutput struct {
	Title          string     `json:"title" jsonschema:"description=带Emoji的小红书标题"`
	Content        string     `json:"content" jsonschema:"description=完整小红书笔记正文"`
	PictureOrder   StringList `json:"picture_order" jsonschema:"description=按发布顺序排列的图片image_id列表"`
	HighlightHooks StringList `json:"highlight_hooks" jsonschema:"description=文中关键钩子句，便于A/B测试"`
}

type SEOOptimizedNote struct {
	OptimizationSummary   string     `json:"optimization_summary" jsonschema:"description=本次SEO优化的要点与改动说明"`
	OptimizedTitle        string     `json:"optimized_title" jsonschema:"description=SEO优化后的标题"`
	OptimizedContent      string     `json:"optimized_content" jsonschema:"description=SEO优化后的正文"`
	OptimizedPictureOrder StringList `json:"optimized_picture_order" jsonschema:"description=结合搜索与转化优化后的图片顺序"`
	Tags                  StringList `json:"tags" jsonschema:"description=5-8个用于搜索与话题分发的标签"`
}

type ContentResult struct {
	Strategy    *ContentStrategyBrief
	Copywriting *CopywritingOutput
	SEO         *SEOOptimizedNote
}

const (
	sessionIdeaText      = "idea_text"
	sessionVisualReport  = "visual_report"
	sessionEditReport    = "edit_report"
	sessionStrategyBrief = "strategy_brief"
	sessionCopywriting   = "copywriting"
	sessionSEOReport     = "seo_report"
)
