package manager

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fitmind/backend/pkg/model"
	"strings"
)

var ErrAIRecognitionResultNotFound = errors.New("ai recognition result not found")

// AIRecognitionManager 负责 public.ai_recognition_results 表的数据访问。
// 只做数据库读写，不负责调用 AI，也不负责业务校验。
type AIRecognitionManager struct {
	db *sql.DB
}

// CreateAIRecognitionResultParams 是创建 AI 识别记录时需要的参数。
// ResultJSON 保存 AI 原始结构化结果，应该是合法 JSON。
type CreateAIRecognitionResultParams struct {
	UserID         string
	ClothingItemID *string
	ImageURL       string
	Provider       string
	ModelName      string
	RequestPrompt  string
	ResultJSON     json.RawMessage
	Confidence     *float64
	Status         model.AIRecognitionStatus
	ErrorMessage   string
}

// LinkAIRecognitionClothingItemParams 用于在用户确认保存衣物后，
// 把识别记录和最终 clothing_items 记录关联起来。
type LinkAIRecognitionClothingItemParams struct {
	ID             string
	UserID         string
	ClothingItemID string
	Status         model.AIRecognitionStatus
}

type aiRecognitionScanner interface {
	Scan(dest ...any) error
}

const aiRecognitionResultFields = `
	id::text,
	user_id::text,
	COALESCE(clothing_item_id::text, ''),
	image_url,
	COALESCE(provider,''),
	COALESCE(model_name, ''),
	COALESCE(request_prompt, ''),
	COALESCE(result_json::text, '{}'),
	confidence,
	status,
	COALESCE(error_message, ''),
	created_at
`

func NewAIRecognitionManager(db *sql.DB) *AIRecognitionManager {
	return &AIRecognitionManager{db: db}
}

func (manager *AIRecognitionManager) Create(ctx context.Context, params CreateAIRecognitionResultParams) (*model.AIRecognitionResult, error) {
	resultJSON := params.ResultJSON
	if len(resultJSON) == 0 {
		resultJSON = json.RawMessage(`{}`)
	}
	if !json.Valid(resultJSON) {
		return nil, errors.New("ai recognition result json is invalid")
	}

	status := params.Status
	if status == "" {
		status = model.AIRecognitionStatusSucceeded
	}

	row := manager.db.QueryRowContext(
		ctx,
		`INSERT INTO public.ai_recognition_results (
			user_id, clothing_item_id, image_url, provider, model_name, request_prompt, result_json, confidence, status, error_message
		) VALUES (
			$1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7::jsonb, $8, $9, NULLIF($10, '')
		) RETURNING`+aiRecognitionResultFields,
		params.UserID,
		params.ClothingItemID,
		params.ImageURL,
		params.Provider,
		params.ModelName,
		params.RequestPrompt,
		string(resultJSON),
		params.Confidence,
		string(status),
		params.ErrorMessage,
	)

	return scanAIRecognitionResult(row)
}

func (manager *AIRecognitionManager) GetByID(ctx context.Context, userID, id string) (*model.AIRecognitionResult, error) {
	row := manager.db.QueryRowContext(
		ctx,
		`SELECT `+aiRecognitionResultFields+`
			FROM public.ai_recognition_results
			WHERE id =$1 AND user_id = $2
			LIMIT 1`,
		id,
		userID,
	)

	return scanAIRecognitionResult(row)
}

func (manager *AIRecognitionManager) ListByUser(ctx context.Context, userID string, limit int) ([]model.AIRecognitionResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := manager.db.QueryContext(
		ctx,
		`SELECT `+aiRecognitionResultFields+`
			FROM public.ai_recognition_results
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT $2`,
		userID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]model.AIRecognitionResult, 0)
	for rows.Next() {
		result, err := scanAIRecognitionResult(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (manager *AIRecognitionManager) LinkClothingItem(ctx context.Context, params LinkAIRecognitionClothingItemParams) (*model.AIRecognitionResult, error) {
	status := params.Status
	if status == "" {
		status = model.AIRecognitionStatusEdited
	}

	row := manager.db.QueryRowContext(
		ctx,
		`UPDATE public.ai_recognition_results
			SET clothing_item_id = $3, status = $4
			WHERE id = $1 AND user_id = $2
			RETURNING `+aiRecognitionResultFields,
		params.ID,
		params.UserID,
		params.ClothingItemID,
		string(status),
	)

	return scanAIRecognitionResult(row)
}

func scanAIRecognitionResult(scanner aiRecognitionScanner) (*model.AIRecognitionResult, error) {
	var result model.AIRecognitionResult
	var clothingItemID string
	var resultJSON string
	var confidence sql.NullFloat64

	err := scanner.Scan(
		&result.ID,
		&result.UserID,
		&clothingItemID,
		&result.ImageURL,
		&result.Provider,
		&result.ModelName,
		&result.RequestPrompt,
		&resultJSON,
		&confidence,
		&result.Status,
		&result.ErrorMessage,
		&result.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAIRecognitionResultNotFound
	}
	if err != nil {
		return nil, err
	}

	clothingItemID = strings.TrimSpace(clothingItemID)
	if clothingItemID != "" {
		result.ClothingItemID = &clothingItemID
	}

	if strings.TrimSpace(resultJSON) == "" {
		resultJSON = "{}"
	}
	result.ResultJSON = json.RawMessage(resultJSON)

	if confidence.Valid {
		value := confidence.Float64
		result.Confidence = &value
	}

	return &result, nil
}

var _ aiRecognitionScanner = (*sql.Row)(nil)
var _ aiRecognitionScanner = (*sql.Rows)(nil)
