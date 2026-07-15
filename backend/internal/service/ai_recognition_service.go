package service

import (
	"context"
	"encoding/json"
	"errors"
	"fitmind/backend/pkg/header/dto"
	"fitmind/backend/pkg/manager"
	"fitmind/backend/pkg/model"
	"strings"
)

type AIRecognitionErrorKind string

const (
	AIRecognitionKindInvalidInput AIRecognitionErrorKind = "invalid_input"
	AIRecognitionKindUnauthorized AIRecognitionErrorKind = "unauthorized"
	AIRecognitionKindNotFound     AIRecognitionErrorKind = "not_found"
	AIRecognitionKindInternal     AIRecognitionErrorKind = "internal"

	localAIRecognitionProvider = "fitmind-local"
	localAIRecognitionModel    = "rule-placeholder-v1"
)

type AIRecognitionError struct {
	Kind    AIRecognitionErrorKind
	Message string
	Err     error
}

func (err *AIRecognitionError) Error() string {
	if err.Err != nil {
		return err.Message
	}
	return err.Message + ": " + err.Err.Error()
}

func (err *AIRecognitionError) Unwrap() error {
	return err.Err
}

type AIRecognitionService struct {
	recognitions *manager.AIRecognitionManager
}

func NewAIRecognitionService(recognitions *manager.AIRecognitionManager) *AIRecognitionService {
	return &AIRecognitionService{
		recognitions: recognitions,
	}
}

func (service *AIRecognitionService) Recognize(ctx context.Context, userID string, request dto.RecognizeClothingRequest) (*dto.AIRecognitionResponse, error) {
	if service == nil || service.recognitions == nil {
		return nil, newAIRecognitionError(AIRecognitionKindInternal, "AI识别服务未初始化", nil)
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, newAIRecognitionError(AIRecognitionKindUnauthorized, "未登录", nil)
	}

	imageURL, err := normalizeAIRecognitionImageURL(request.ImageURL)
	if err != nil {
		return nil, err
	}

	prompt := normalizeAIRecognitionPrompt(request.Prompt)

	result := recognizeClothingLocally(imageURL, prompt)

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, newAIRecognitionError(AIRecognitionKindInternal, "生成识别结果失败", err)
	}

	record, err := service.recognitions.Create(ctx, manager.CreateAIRecognitionResultParams{
		UserID:        userID,
		ImageURL:      imageURL,
		Provider:      localAIRecognitionProvider,
		ModelName:     localAIRecognitionModel,
		RequestPrompt: prompt,
		ResultJSON:    resultJSON,
		Confidence:    result.AIConfidence,
		Status:        model.AIRecognitionStatusSucceeded,
	})
	if err != nil {
		return nil, newAIRecognitionError(AIRecognitionKindInternal, "保存AI识别记录失败", err)
	}

	response, err := toAIRecognitionResponse(record)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (service *AIRecognitionService) GetByID(ctx context.Context, userID, id string) (*dto.AIRecognitionResponse, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, newAIRecognitionError(AIRecognitionKindUnauthorized, "未登录", nil)
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, newAIRecognitionError(AIRecognitionKindInvalidInput, "识别记录ID不能为空", nil)
	}

	record, err := service.recognitions.GetByID(ctx, userID, id)
	if errors.Is(err, manager.ErrAIRecognitionResultNotFound) {
		return nil, newAIRecognitionError(AIRecognitionKindNotFound, "识别记录不存在", err)
	}
	if err != nil {
		return nil, newAIRecognitionError(AIRecognitionKindInternal, "查询识别记录失败", err)
	}

	response, err := toAIRecognitionResponse(record)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (service *AIRecognitionManager) ListByUser(ctx context.Context, userID string, limit int) ([]dto.AIRecognitionResponse, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, newAIRecognitionError(AIRecognitionKindUnauthorized, "未登录", nil)
	}

	records, err := service.recognitions.ListByUser(ctx, userID, limit)
	if err != nil {
		return nil, newAIRecognitionError(AIRecognitionKindInternal, "查询识别记录列表失败", err)
	}

	responses := make([]dto.AIRecognitionResponse, 0, len(records))
	for index := range records {
		response, err := toAIRecognitionResponse(&records[index])
		if err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}

	return responses, nil
}

func (service *AIRecognitionService) LinkClothingItem(ctx context.Context, userID, recognitionID, clothingItemID string) (*dto.AIRecognitionResponse, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, newAIRecognitionError(AIRecognitionKindUnauthorized, "未登录", nil)
	}

	recognitionID = strings.TrimSpace(recognitionID)
	if recognitionID == "" {
		return nil, newAIRecognitionError(AIRecognitionKindInvalidInput, "识别记录ID不能为空", nil)
	}

	clothingItemID = strings.TrimSpace(clothingItemID)
	if clothingItemID == "" {
		return nil, newAIRecognitionError(AIRecognitionKindInvalidInput, "衣物ID不能为空", nil)
	}

	record, err := service.recognitions.LinkClothingItem(ctx, manager.LinkAIRecognitionClothingItemParams{
		ID:             recognitionID,
		UserID:         userID,
		ClothingItemID: clothingItemID,
		Status:         model.AIRecognitionStatusEdited,
	})
	if errors.Is(err, manager.ErrAIRecognitionResultNotFound) {
		return nil, newAIRecognitionError(AIRecognitionKindNotFound, "识别记录不存在", err)
	}
	if err != nil {
		return nil, newAIRecognitionError(AIRecognitionKindInternal, "关联识别记录失败", err)
	}

	response, err := toAIRecognitionResponse(record)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func AIRecognitionKindOf(err error) AIRecognitionErrorKind {
	var recognitionErr *AIRecognitionError
	if errors.As(err, &recognitionErr) {
		return recognitionErr.Kind
	}
	return AIRecognitionKindInternal
}

func AIRecognitionMessageOf(err error) string {
	var recognitionErr *AIRecognitionError
	if errors.As(err, &recognitionErr) {
		return recognitionErr.Message
	}
	return "服务器内部错误"
}

func normalizeAIRecognitionImageURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", newAIRecognitionError(AIRecognitionKindInvalidInput, "衣物图片不能为空", nil)
	}
	if len([]rune(value)) > 2048 {
		return "", newAIRecognitionError(AIRecognitionKindInvalidInput, "衣物图片不能超过2048个字符", nil)
	}
	return value, nil
}

func normalizeAIRecognitionPrompt(value string) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > 500 {
		return string([]rune(value)[:500])
	}
	return value
}

func recognizeClothingLocally(imageURL, prompt string) dto.RecognizedClothingTags {}

func detectAIRecognitionCategory(text string) string {
	switch {
	case aiContainsAny(text, "连衣裙", "dress"):
		return "dress"
	case aiContainsAny(text, "外套", "西装", "大衣", "风衣", "")
	}
}

func aiContainsAny(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func aiStringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func intPtr(value int) *int {
	return &value
}

func floatPtr(value float64) *float64 {
	return &value
}

func newAIRecognitionError(kind AIRecognitionErrorKind, message string, err error) *AIRecognitionError {
	return &AIRecognitionError{
		Kind:    kind,
		Message: message,
		Err:     err,
	}
}
