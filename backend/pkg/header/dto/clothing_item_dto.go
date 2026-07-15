package dto

// CreateClothingItemRequest 是创建衣物接口的请求体。
// 输入：用户确认后的衣物图片、类别、颜色、季节、风格等结构化信息。
// 输出：交给 service 校验后创建 clothing_items 记录。
type CreateClothingItemRequest struct {
	Name           string   `json:"name"`
	ImageURL       string   `json:"imageUrl"`
	ThumbnailURL   string   `json:"thumbnailUrl"`
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
	ActivityLevel  *int     `json:"activityLevel"`
	AIConfidence   *float64 `json:"aiConfidence"`
	Notes          string   `json:"notes"`
}

// UpdateClothingItemRequest 是更新衣物接口的请求体。
// 规则：指针字段用于区分“未传入”和“传入空值”。
// 例：name 为 nil 表示不修改；name 指向空字符串表示尝试把名称改为空，会被 service 拦截。
type UpdateClothingItemRequest struct {
	Name           *string   `json:"name"`
	ImageURL       *string   `json:"imageUrl"`
	ThumbnailURL   *string   `json:"thumbnailUrl"`
	Category       *string   `json:"category"`
	SubCategory    *string   `json:"subCategory"`
	ColorMain      *string   `json:"colorMain"`
	ColorSecondary *string   `json:"colorSecondary"`
	SeasonTags     *[]string `json:"seasonTags"`
	StyleTags      *[]string `json:"styleTags"`
	Material       *string   `json:"material"`
	Thickness      *string   `json:"thickness"`
	FitType        *string   `json:"fitType"`
	FormalityScore *int      `json:"formalityScore"`
	ActivityLevel  *int      `json:"activityLevel"`
	Status         *string   `json:"status"`
	AIConfidence   *float64  `json:"aiConfidence"`
	Notes          *string   `json:"notes"`
}

// ClothingItemResponse 是衣物接口返回给前端的数据结构。
// 输出：隐藏数据库内部字段 deleted_at，并把时间统一转为 RFC3339 字符串。
type ClothingItemResponse struct {
	ID             string   `json:"id"`
	UserID         string   `json:"userId"`
	Name           string   `json:"name"`
	ImageURL       string   `json:"imageUrl"`
	ThumbnailURL   string   `json:"thumbnailUrl"`
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
	ActivityLevel  *int     `json:"activityLevel"`
	Status         string   `json:"status"`
	WearCount      int      `json:"wearCount"`
	LastWornAt     string   `json:"lastWornAt"`
	AIConfidence   *float64 `json:"aiConfidence"`
	Notes          string   `json:"notes"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
}
