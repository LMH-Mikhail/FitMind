package manager

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fitmind/backend/pkg/model"
	"fmt"
	"strings"
)

// ErrClothingItemNotFound 表示指定衣物不存在，或不属于当前用户。
// 作用：service 根据这个错误返回 404，而不是 500。
var ErrClothingItemNotFound = errors.New("clothing item not found")

// ClothingItemManager 负责 clothing_items 表的数据访问。
// 输入：service 传入的业务参数。
// 输出：model.ClothingItem 或数据库错误。
type ClothingItemManager struct {
	db *sql.DB
}

// CreateClothingItemParams 是创建衣物时 manager 层需要的完整参数。
// UserID 必须来自认证上下文，不能从前端请求体直接相信。
type CreateClothingItemParams struct {
	UserID         string
	Name           string
	ImageURL       string
	ThumbnailURL   string
	Category       string
	SubCategory    string
	ColorMain      string
	ColorSecondary string
	SeasonTags     []string
	StyleTags      []string
	Material       string
	Thickness      string
	FitType        string
	FormalityScore *int
	ActivityLevel  *int
	AIConfidence   *float64
	Notes          string
}

// UpdateClothingItemParams 是更新衣物时 manager 层使用的完整快照。
// 说明：service 会先读取旧记录，再合并请求字段，最后传入完整参数。
type UpdateClothingItemParams struct {
	ID             string
	UserID         string
	Name           string
	ImageURL       string
	ThumbnailURL   string
	Category       string
	SubCategory    string
	ColorMain      string
	ColorSecondary string
	SeasonTags     []string
	StyleTags      []string
	Material       string
	Thickness      string
	FitType        string
	FormalityScore *int
	ActivityLevel  *int
	Status         string
	AIConfidence   *float64
	Notes          string
}

// ListClothingItemsFilter 是衣物列表查询条件。
// PageSize 和 Offset 由 service 使用 model.NewPagination 计算得到。
type ListClothingItemsFilter struct {
	UserID   string
	Category string
	Status   string
	PageSize int
	Offset   int
}

type clothingItemScanner interface {
	Scan(dest ...any) error
}

const clothingItemFields = `
	id::text,
	user_id::text,
	name,
	image_url,
	COALESCE(thumbnail_url, ''),
	category,
	COALESCE(sub_category, ''),
	COALESCE(color_main, ''),
	COALESCE(color_secondary, ''),
	COALESCE(to_jsonb(season_tags)::text, '[]'),
	COALESCE(to_jsonb(style_tags)::text, '[]'),
	COALESCE(material, ''),
	thickness,
	fit_type,
	formality_score,
	activity_level,
	status,
	wear_count,
	last_worn_at,
	ai_confidence,
	COALESCE(notes, ''),
	created_at,
	updated_at
`

func NewClothingItemManager(db *sql.DB) *ClothingItemManager {
	return &ClothingItemManager{db: db}
}

func (manager *ClothingItemManager) Create(ctx context.Context, params CreateClothingItemParams) (*model.ClothingItem, error) {
	seasonTagsJSON, err := marshalStringSlice(params.SeasonTags)
	if err != nil {
		return nil, err
	}

	styleTagsJSON, err := marshalStringSlice(params.StyleTags)
	if err != nil {
		return nil, err
	}

	row := manager.db.QueryRowContext(
		ctx,
		`INSERT INTO public.clothing_items (
			user_id,
			name,
			image_url,
			thumbnail_url,
			category,
			sub_category,
			color_main,
			color_secondary,
			season_tags,
			style_tags,
			material,
			thickness,
			fit_type,
			formality_score,
			activity_level,
			ai_confidence,
			notes
		)
		VALUES (
			$1,
			$2,
			$3,
			NULLIF($4, ''),
			$5,
			NULLIF($6, ''),
			NULLIF($7, ''),
			NULLIF($8, ''),
			ARRAY(SELECT jsonb_array_elements_text($9::jsonb)),
			ARRAY(SELECT jsonb_array_elements_text($10::jsonb)),
			NULLIF($11, ''),
			$12,
			$13,
			$14,
			$15,
			$16,
			NULLIF($17, '')
		)
		RETURNING `+clothingItemFields,
		params.UserID,
		params.Name,
		params.ImageURL,
		params.ThumbnailURL,
		params.Category,
		params.SubCategory,
		params.ColorMain,
		params.ColorSecondary,
		seasonTagsJSON,
		styleTagsJSON,
		params.Material,
		params.Thickness,
		params.FitType,
		params.FormalityScore,
		params.ActivityLevel,
		params.AIConfidence,
		params.Notes,
	)

	return scanClothingItem(row)
}

func (manager *ClothingItemManager) GetByID(ctx context.Context, userID, id string) (*model.ClothingItem, error) {
	row := manager.db.QueryRowContext(
		ctx,
		`SELECT `+clothingItemFields+`
		FROM public.clothing_items
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		LIMIT 1`,
		id,
		userID,
	)

	return scanClothingItem(row)
}

func (manager *ClothingItemManager) List(ctx context.Context, filter ListClothingItemsFilter) ([]model.ClothingItem, int, error) {
	conditions := []string{"user_id = $1", "deleted_at IS NULL"}
	args := []any{filter.UserID}
	nextArg := 2

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", nextArg))
		args = append(args, filter.Category)
		nextArg++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", nextArg))
		args = append(args, filter.Status)
		nextArg++
	}

	whereSQL := strings.Join(conditions, " AND ")

	var total int
	countQuery := `SELECT COUNT(*) FROM public.clothing_items WHERE ` + whereSQL
	if err := manager.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, filter.PageSize, filter.Offset)

	query := fmt.Sprintf(
		`SELECT %s
		FROM public.clothing_items
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		clothingItemFields,
		whereSQL,
		nextArg,
		nextArg+1,
	)

	rows, err := manager.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]model.ClothingItem, 0)
	for rows.Next() {
		item, err := scanClothingItem(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (manager *ClothingItemManager) Update(ctx context.Context, params UpdateClothingItemParams) (*model.ClothingItem, error) {
	seasonTagsJSON, err := marshalStringSlice(params.SeasonTags)
	if err != nil {
		return nil, err
	}

	styleTagsJSON, err := marshalStringSlice(params.StyleTags)
	if err != nil {
		return nil, err
	}

	row := manager.db.QueryRowContext(
		ctx,
		`UPDATE public.clothing_items
		SET
			name = $3,
			image_url = $4,
			thumbnail_url = NULLIF($5, ''),
			category = $6,
			sub_category = NULLIF($7, ''),
			color_main = NULLIF($8, ''),
			color_secondary = NULLIF($9, ''),
			season_tags = ARRAY(SELECT jsonb_array_elements_text($10::jsonb)),
			style_tags = ARRAY(SELECT jsonb_array_elements_text($11::jsonb)),
			material = NULLIF($12, ''),
			thickness = $13,
			fit_type = $14,
			formality_score = $15,
			activity_level = $16,
			status = $17,
			ai_confidence = $18,
			notes = NULLIF($19, ''),
			updated_at = now()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		RETURNING `+clothingItemFields,
		params.ID,
		params.UserID,
		params.Name,
		params.ImageURL,
		params.ThumbnailURL,
		params.Category,
		params.SubCategory,
		params.ColorMain,
		params.ColorSecondary,
		seasonTagsJSON,
		styleTagsJSON,
		params.Material,
		params.Thickness,
		params.FitType,
		params.FormalityScore,
		params.ActivityLevel,
		params.Status,
		params.AIConfidence,
		params.Notes,
	)

	return scanClothingItem(row)
}

func (manager *ClothingItemManager) SoftDelete(ctx context.Context, userID, id string) error {
	result, err := manager.db.ExecContext(
		ctx,
		`UPDATE public.clothing_items
		SET status = 'deleted', deleted_at = now(), updated_at = now()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		id,
		userID,
	)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrClothingItemNotFound
	}

	return nil
}

func scanClothingItem(scanner clothingItemScanner) (*model.ClothingItem, error) {
	var item model.ClothingItem
	var formalityScore sql.NullInt64
	var activityLevel sql.NullInt64
	var lastWornAt sql.NullTime
	var aiConfidence sql.NullFloat64
	var seasonTagsJSON string
	var styleTagsJSON string

	err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.Name,
		&item.ImageURL,
		&item.ThumbnailURL,
		&item.Category,
		&item.SubCategory,
		&item.ColorMain,
		&item.ColorSecondary,
		&seasonTagsJSON,
		&styleTagsJSON,
		&item.Material,
		&item.Thickness,
		&item.FitType,
		&formalityScore,
		&activityLevel,
		&item.Status,
		&item.WearCount,
		&lastWornAt,
		&aiConfidence,
		&item.Notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrClothingItemNotFound
	}
	if err != nil {
		return nil, err
	}

	if formalityScore.Valid {
		value := int(formalityScore.Int64)
		item.FormalityScore = &value
	}

	if activityLevel.Valid {
		value := int(activityLevel.Int64)
		item.ActivityLevel = &value
	}

	if lastWornAt.Valid {
		item.LastWornAt = &lastWornAt.Time
	}

	if aiConfidence.Valid {
		value := aiConfidence.Float64
		item.AIConfidence = &value
	}

	if err = json.Unmarshal([]byte(seasonTagsJSON), &item.SeasonTags); err != nil {
		return nil, err
	}

	if err = json.Unmarshal([]byte(styleTagsJSON), &item.StyleTags); err != nil {
		return nil, err
	}

	return &item, nil
}

func marshalStringSlice(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}

	data, err := json.Marshal(values)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

var _ clothingItemScanner = (*sql.Row)(nil)
var _ clothingItemScanner = (*sql.Rows)(nil)
