package model

import "time"

// BaseModel是所有数据库表模型的通用字段
// ID：自增主键
// CreatedAt：创建时间，由数据库或插入逻辑写入
// UpdatedAt：更新时间，由数据库或更新逻辑写入
// DeletedAt：软删除时间；nil表示未删除
type BaseModel struct {
	ID        int64      `json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"-"`
}

// Pagination保存列表查询的分页参数
// PageNum：当前页码，从1开始
// PageSize：每页数量
// Offset：SQL查询偏移量
type Pagination struct {
	PageNum  int
	PageSize int
	Offset   int
}

// NewPagination根据输入构造安全分页参数
// 输入：
// - pageNum：前端传入页码
// - pageSize：前端传入每页数量
// 输出：归一化后的Pagination
// 规则：
// - pageNum小于1时重置为1
// - pageSize小于1时重置为20
// - pageSize大于100时限制为100
// - Offset = (PageNum - 1) * PageSize
func NewPagination(pageNum int, pageSize int) Pagination {
	if pageNum < 1 {
		pageNum = 1
	}

	if pageSize < 1 {
		pageSize = 20
	}

	if pageSize > 100 {
		pageSize = 100
	}

	return Pagination{
		PageNum:  pageNum,
		PageSize: pageSize,
		Offset:   (pageNum - 1) * pageSize,
	}
}
