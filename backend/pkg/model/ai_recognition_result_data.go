package model

import (
	"encoding/json"
	"time"
)

// AIRecognitionStatus 表示 AI 识别记录的处理状态。
// 取值需要和 schema.sql 里的 ai_recognition_status_check 保持一致。
type AIRecognitionStatus string

const (
	AIRecognitionStatusPending   AIRecognitionStatus = "pending"
	AIRecognitionStatusSucceeded AIRecognitionStatus = "succeeded"
	AIRecognitionStatusFailed    AIRecognitionStatus = "failed"
	AIRecognitionStatusEdited    AIRecognitionStatus = "edited"
)

// AIRecognitionResult 对应数据库 public.ai_recognition_results 表。
// 作用：保存 AI 原始识别结果，和用户最终确认后的 clothing_items 数据分开。
type AIRecognitionResult struct {
	ID             string
	UserID         string
	ClothingItemID *string
	ImageURL       string
	Provider       string
	ModelName      string
	RequestPrompt  string
	ResultJSON     json.RawMessage
	Confidence     *float64
	Status         AIRecognitionStatus
	ErrorMessage   string
	CreatedAt      time.Time
}
