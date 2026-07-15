package service

import (
	"context"
	"errors"
	"fitmind/backend/pkg/header/common"
	"fitmind/backend/pkg/header/dto"
	"fitmind/backend/pkg/manager"
	"fitmind/backend/pkg/model"
	"fmt"
	"strings"
	"time"
)

type ClothingErrorKind string

const (
	ClothingKindInvalidInput ClothingErrorKind = "invalid_input"
	ClothingKindNotFound     ClothingErrorKind = "not_found"
	ClothingKindInternal     ClothingErrorKind = "internal"
)

type ClothingError struct {
	Kind    ClothingErrorKind
	Message string
	Err     error
}

func (err *ClothingError) Error() string {
	if err.Err == nil {
		return err.Message
	}
	return err.Message + ": " + err.Err.Error()
}

func (err *ClothingError) Unwrap() error {
	return err.Err
}

type ClothingItemService struct {
	items *manager.ClothingItemManager
}

func NewClothingItemService(items *manager.ClothingItemManager) *ClothingItemService {
	return &ClothingItemService{items: items}
}

func (service *ClothingItemService) Create(ctx context.Context, userID string, request dto.CreateClothingItemRequest) (*dto.ClothingItemResponse, error) {
	params, err := normalizeCreateRequest(userID, request)
	if err != nil {
		return nil, err
	}

	item, err := service.items.Create(ctx, params)
	if err != nil {
		return nil, newClothingError(ClothingKindInternal, "创建衣物失败", err)
	}

	response := toClothingItemResponse(item)
	return &response, nil
}

func (service *ClothingItemService) List(ctx context.Context, userID string, pageNum, pageSize int, category, status string) (*common.Page, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, newClothingError(ClothingKindInvalidInput, "未登录", nil)
	}

	category = strings.TrimSpace(category)
	if category != "" && !isAllowedCategory(category) {
		return nil, newClothingError(ClothingKindInvalidInput, "衣物类别不合法", nil)
	}

	status = strings.TrimSpace(status)
	if status != "" && !isAllowedStatus(status) {
		return nil, newClothingError(ClothingKindInvalidInput, "衣物状态不合法", nil)
	}

	pagination := model.NewPagination(pageNum, pageSize)

	items, total, err := service.items.List(ctx, manager.ListClothingItemsFilter{
		UserID:   userID,
		Category: category,
		Status:   status,
		PageSize: pagination.PageSize,
		Offset:   pagination.Offset,
	})
	if err != nil {
		return nil, newClothingError(ClothingKindInternal, "查询衣物列表失败", err)
	}

	list := make([]dto.ClothingItemResponse, 0, len(items))
	for index := range items {
		item := items[index]
		list = append(list, toClothingItemResponse(&item))
	}

	totalPage := 0
	if total > 0 {
		totalPage = (total + pagination.PageSize - 1) / pagination.PageSize
	}

	return &common.Page{
		PageNum:   pagination.PageNum,
		PageSize:  pagination.PageSize,
		TotalPage: totalPage,
		Total:     total,
		List:      list,
	}, nil
}

func (service *ClothingItemService) GetByID(ctx context.Context, userID, id string) (*dto.ClothingItemResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, newClothingError(ClothingKindInvalidInput, "衣物ID不能为空", nil)
	}

	item, err := service.items.GetByID(ctx, userID, id)
	if errors.Is(err, manager.ErrClothingItemNotFound) {
		return nil, newClothingError(ClothingKindNotFound, "衣物不存在", err)
	}
	if err != nil {
		return nil, newClothingError(ClothingKindInternal, "查询衣物失败", err)
	}

	response := toClothingItemResponse(item)
	return &response, nil
}

func (service *ClothingItemService) Update(ctx context.Context, userID, id string, request dto.UpdateClothingItemRequest) (*dto.ClothingItemResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, newClothingError(ClothingKindInvalidInput, "衣物ID不能为空", nil)
	}

	current, err := service.items.GetByID(ctx, userID, id)
	if errors.Is(err, manager.ErrClothingItemNotFound) {
		return nil, newClothingError(ClothingKindNotFound, "衣物不存在", err)
	}
	if err != nil {
		return nil, newClothingError(ClothingKindInternal, "查询衣物失败", err)
	}

	merged, err := mergeUpdateRequest(current, request)
	if err != nil {
		return nil, err
	}

	updated, err := service.items.Update(ctx, merged)
	if errors.Is(err, manager.ErrClothingItemNotFound) {
		return nil, newClothingError(ClothingKindNotFound, "衣物不存在", err)
	}
	if err != nil {
		return nil, newClothingError(ClothingKindInternal, "更新衣物失败", err)
	}

	response := toClothingItemResponse(updated)
	return &response, nil
}

func (service *ClothingItemService) Delete(ctx context.Context, userID, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return newClothingError(ClothingKindInvalidInput, "衣物ID不能为空", nil)
	}

	err := service.items.SoftDelete(ctx, userID, id)
	if errors.Is(err, manager.ErrClothingItemNotFound) {
		return newClothingError(ClothingKindNotFound, "衣物不存在", err)
	}
	if err != nil {
		return newClothingError(ClothingKindInternal, "删除衣物失败", err)
	}

	return nil
}

func ClothingKindOf(err error) ClothingErrorKind {
	var clothingErr *ClothingError
	if errors.As(err, &clothingErr) {
		return clothingErr.Kind
	}
	return ClothingKindInternal
}

func ClothingMessageOf(err error) string {
	var clothingErr *ClothingError
	if errors.As(err, &clothingErr) {
		return clothingErr.Message
	}
	return "服务器内部错误"
}

func normalizeCreateRequest(userID string, request dto.CreateClothingItemRequest) (manager.CreateClothingItemParams, error) {
	if strings.TrimSpace(userID) == "" {
		return manager.CreateClothingItemParams{}, newClothingError(ClothingKindInvalidInput, "未登录", nil)
	}

	name, err := normalizeRequiredString(request.Name, "衣物名称", 80)
	if err != nil {
		return manager.CreateClothingItemParams{}, err
	}

	imageURL, err := normalizeRequiredString(request.ImageURL, "衣物图片", 2048)
	if err != nil {
		return manager.CreateClothingItemParams{}, err
	}

	category, err := normalizeCategory(request.Category)
	if err != nil {
		return manager.CreateClothingItemParams{}, err
	}

	thickness := normalizeDefaultString(request.Thickness, "unknown")
	if !isAllowedThickness(thickness) {
		return manager.CreateClothingItemParams{}, newClothingError(ClothingKindInvalidInput, "厚薄程度不合法", nil)
	}

	fitType := normalizeDefaultString(request.FitType, "unknown")
	if !isAllowedFitType(fitType) {
		return manager.CreateClothingItemParams{}, newClothingError(ClothingKindInvalidInput, "版型不合法", nil)
	}

	if err = validateRangePointer(request.FormalityScore, "正式程度"); err != nil {
		return manager.CreateClothingItemParams{}, err
	}

	if err = validateRangePointer(request.ActivityLevel, "活动程度"); err != nil {
		return manager.CreateClothingItemParams{}, err
	}

	if err = validateConfidence(request.AIConfidence); err != nil {
		return manager.CreateClothingItemParams{}, err
	}

	return manager.CreateClothingItemParams{
		UserID:         userID,
		Name:           name,
		ImageURL:       imageURL,
		ThumbnailURL:   normalizeOptionalString(request.ThumbnailURL, 2048),
		Category:       category,
		SubCategory:    normalizeOptionalString(request.SubCategory, 80),
		ColorMain:      normalizeOptionalString(request.ColorMain, 40),
		ColorSecondary: normalizeOptionalString(request.ColorSecondary, 40),
		SeasonTags:     normalizeStringSlice(request.SeasonTags, 12),
		StyleTags:      normalizeStringSlice(request.StyleTags, 12),
		Material:       normalizeOptionalString(request.Material, 80),
		Thickness:      thickness,
		FitType:        fitType,
		FormalityScore: request.FormalityScore,
		ActivityLevel:  request.ActivityLevel,
		AIConfidence:   request.AIConfidence,
		Notes:          normalizeOptionalString(request.Notes, 500),
	}, nil
}

func mergeUpdateRequest(current *model.ClothingItem, request dto.UpdateClothingItemRequest) (manager.UpdateClothingItemParams, error) {
	params := manager.UpdateClothingItemParams{
		ID:             current.ID,
		UserID:         current.UserID,
		Name:           current.Name,
		ImageURL:       current.ImageURL,
		ThumbnailURL:   current.ThumbnailURL,
		Category:       current.Category,
		SubCategory:    current.SubCategory,
		ColorMain:      current.ColorMain,
		ColorSecondary: current.ColorSecondary,
		SeasonTags:     current.SeasonTags,
		StyleTags:      current.StyleTags,
		Material:       current.Material,
		Thickness:      current.Thickness,
		FitType:        current.FitType,
		FormalityScore: current.FormalityScore,
		ActivityLevel:  current.ActivityLevel,
		Status:         current.Status,
		AIConfidence:   current.AIConfidence,
		Notes:          current.Notes,
	}

	var err error

	if request.Name != nil {
		params.Name, err = normalizeRequiredString(*request.Name, "衣物名称", 80)
		if err != nil {
			return params, err
		}
	}

	if request.ImageURL != nil {
		params.ImageURL, err = normalizeRequiredString(*request.ImageURL, "衣物图片", 2048)
		if err != nil {
			return params, err
		}
	}

	if request.Category != nil {
		params.Category, err = normalizeCategory(*request.Category)
		if err != nil {
			return params, err
		}
	}

	if request.ThumbnailURL != nil {
		params.ThumbnailURL = normalizeOptionalString(*request.ThumbnailURL, 2048)
	}

	if request.SubCategory != nil {
		params.SubCategory = normalizeOptionalString(*request.SubCategory, 80)
	}

	if request.ColorMain != nil {
		params.ColorMain = normalizeOptionalString(*request.ColorMain, 40)
	}

	if request.ColorSecondary != nil {
		params.ColorSecondary = normalizeOptionalString(*request.ColorSecondary, 40)
	}

	if request.SeasonTags != nil {
		params.SeasonTags = normalizeStringSlice(*request.SeasonTags, 12)
	}

	if request.StyleTags != nil {
		params.StyleTags = normalizeStringSlice(*request.StyleTags, 12)
	}

	if request.Material != nil {
		params.Material = normalizeOptionalString(*request.Material, 80)
	}

	if request.Thickness != nil {
		params.Thickness = normalizeDefaultString(*request.Thickness, "unknown")
		if !isAllowedThickness(params.Thickness) {
			return params, newClothingError(ClothingKindInvalidInput, "厚薄程度不合法", nil)
		}
	}

	if request.FitType != nil {
		params.FitType = normalizeDefaultString(*request.FitType, "unknown")
		if !isAllowedFitType(params.FitType) {
			return params, newClothingError(ClothingKindInvalidInput, "版型不合法", nil)
		}
	}

	if request.FormalityScore != nil {
		if err = validateRangePointer(request.FormalityScore, "正式程度"); err != nil {
			return params, err
		}
		params.FormalityScore = request.FormalityScore
	}

	if request.ActivityLevel != nil {
		if err = validateRangePointer(request.ActivityLevel, "活动程度"); err != nil {
			return params, err
		}
		params.ActivityLevel = request.ActivityLevel
	}

	if request.Status != nil {
		status := strings.TrimSpace(*request.Status)
		if !isAllowedStatus(status) || status == "deleted" {
			return params, newClothingError(ClothingKindInvalidInput, "衣物状态不合法", nil)
		}
		params.Status = status
	}

	if request.AIConfidence != nil {
		if err = validateConfidence(request.AIConfidence); err != nil {
			return params, err
		}
		params.AIConfidence = request.AIConfidence
	}

	if request.Notes != nil {
		params.Notes = normalizeOptionalString(*request.Notes, 500)
	}

	return params, nil
}

func toClothingItemResponse(item *model.ClothingItem) dto.ClothingItemResponse {
	lastWornAt := ""
	if item.LastWornAt != nil {
		lastWornAt = item.LastWornAt.Format(time.RFC3339)
	}

	return dto.ClothingItemResponse{
		ID:             item.ID,
		UserID:         item.UserID,
		Name:           item.Name,
		ImageURL:       item.ImageURL,
		ThumbnailURL:   item.ThumbnailURL,
		Category:       item.Category,
		SubCategory:    item.SubCategory,
		ColorMain:      item.ColorMain,
		ColorSecondary: item.ColorSecondary,
		SeasonTags:     item.SeasonTags,
		StyleTags:      item.StyleTags,
		Material:       item.Material,
		Thickness:      item.Thickness,
		FitType:        item.FitType,
		FormalityScore: item.FormalityScore,
		ActivityLevel:  item.ActivityLevel,
		Status:         item.Status,
		WearCount:      item.WearCount,
		LastWornAt:     lastWornAt,
		AIConfidence:   item.AIConfidence,
		Notes:          item.Notes,
		CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
	}
}

func normalizeRequiredString(value, fieldName string, maxLength int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", newClothingError(ClothingKindInvalidInput, fieldName+"不能为空", nil)
	}
	if len([]rune(value)) > maxLength {
		return "", newClothingError(ClothingKindInvalidInput, fmt.Sprintf("%s不能超过%d个字符", fieldName, maxLength), nil)
	}
	return value, nil
}

func normalizeOptionalString(value string, maxLength int) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > maxLength {
		return string([]rune(value)[:maxLength])
	}
	return value
}

func normalizeDefaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func normalizeStringSlice(values []string, maxCount int) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}

		seen[value] = true
		result = append(result, value)

		if len(result) >= maxCount {
			break
		}
	}

	return result
}

func normalizeCategory(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !isAllowedCategory(value) {
		return "", newClothingError(ClothingKindInvalidInput, "衣物类别不合法", nil)
	}
	return value, nil
}

func validateRangePointer(value *int, fieldName string) error {
	if value == nil {
		return nil
	}
	if *value < 1 || *value > 5 {
		return newClothingError(ClothingKindInvalidInput, fieldName+"必须在1到5之间", nil)
	}
	return nil
}

func validateConfidence(value *float64) error {
	if value == nil {
		return nil
	}
	if *value < 0 || *value > 1 {
		return newClothingError(ClothingKindInvalidInput, "AI置信度必须在0到1之间", nil)
	}
	return nil
}

func isAllowedCategory(value string) bool {
	return map[string]bool{
		"top":       true,
		"bottom":    true,
		"outerwear": true,
		"dress":     true,
		"shoes":     true,
		"bag":       true,
		"accessory": true,
	}[value]
}

func isAllowedThickness(value string) bool {
	return map[string]bool{
		"thin":    true,
		"regular": true,
		"thick":   true,
		"unknown": true,
	}[value]
}

func isAllowedFitType(value string) bool {
	return map[string]bool{
		"slim":      true,
		"regular":   true,
		"loose":     true,
		"oversized": true,
		"unknown":   true,
	}[value]
}

func isAllowedStatus(value string) bool {
	return map[string]bool{
		"active":          true,
		"laundry":         true,
		"idle":            true,
		"not_recommended": true,
		"deleted":         true,
	}[value]
}

func newClothingError(kind ClothingErrorKind, message string, err error) *ClothingError {
	return &ClothingError{
		Kind:    kind,
		Message: message,
		Err:     err,
	}
}
