package model

import "time"

// ClothingItem 对应数据库 public.clothing_items 表。
// 作用：承载 manager 从数据库读取出来的衣物数据。
type ClothingItem struct {
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
	WearCount      int
	LastWornAt     *time.Time
	AIConfidence   *float64
	Notes          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
