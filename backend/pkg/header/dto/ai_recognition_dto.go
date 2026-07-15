package dto

// RecognizeClothingRequest 是 AI 衣物识别接口的请求体。
// 输入：图片 URL 来自上传接口返回的 imageUrl；prompt 是用户可选补充描述。
type RecognizeClothingRequest struct {
	ImageURL string `json:"imageUrl"`
	Prompt   string `json:"prompt"`
}

// RecognizedClothingTags 是 AI 识别后返回给前端确认/编辑的结构化衣物标签。
// 这些字段后续可以直接映射到 CreateClothingItemRequest。
type RecognizedClothingTags struct {
	Name           string   `json:"name"`
	ImageURL       string   `json:"imageUrl"`
	Category       string   `json:"category"`
	SubCategory    string   `json:"subCategory"`
	ColorMain      string   `json:"colorMain"`
	ColorSecondary string   `json:"colorSecondary"`
	SeasonTags     []string `json:"seasonTags"`
	StyleTags      []string `json:"styleTags"`
	Material       string   `json:"material"`
	Thickness      string   `json:"thickness"`
	FitType        string   `json:"fitType"`
	FormalityScore *int     `json:"formalityScore"`
	ActivityLevelS *int     `json:"activityLevelScore"`
	AIConfidence   *float64 `json:"aiConfidence"`
}

// AIRecognitionResponse 是 AI 衣物识别接口返回给前端的数据。
// result 是可编辑标签；record 保存本次识别记录的元信息。
type AIRecognitionResponse struct {
	ID           string                 `json:"id"`
	ImageURL     string                 `json:"imageUrl"`
	Provider     string                 `json:"provider"`
	ModelName    string                 `json:"modelName"`
	Status       string                 `json:"status"`
	Confidence   *float64               `json:"confidence"`
	Result       RecognizedClothingTags `json:"result"`
	ErrorMessage string                 `json:"errorMessage"`
	CreatedAt    string                 `json:"createdAt"`
}
