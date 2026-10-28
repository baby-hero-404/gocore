package pagination

import (
	"context"

	"github.com/baby-hero-404/gocore/stderr"
	"gorm.io/gorm"
)

// OffsetRequest is the client input for offset pagination.
type OffsetRequest struct {
	Page     int `form:"page" json:"page"`           // 1-based
	PageSize int `form:"page_size" json:"page_size"` // rows per page
}

// OffsetPage is one page of an offset-paginated result.
type OffsetPage[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// OffsetQuery configures an offset query.
type OffsetQuery struct {
	Sorts           []Sort
	DefaultPageSize int // default 20
	MaxPageSize     int // default 100
}

// Offset returns the requested page of T plus totals. db must already carry filters and joins;
// the total is COUNT(*) over db, so avoid GROUP BY in db (use a subquery instead).
func Offset[T any](ctx context.Context, db *gorm.DB, q OffsetQuery, req OffsetRequest) (OffsetPage[T], error) {
	size := clampLimit(req.PageSize, q.DefaultPageSize, q.MaxPageSize)
	page := max(req.Page, 1)
	result := OffsetPage[T]{Items: []T{}, Page: page, PageSize: size}

	if err := validateSorts(q.Sorts); err != nil {
		return result, err
	}

	base := db.WithContext(ctx).Session(&gorm.Session{})
	if err := base.Model(new(T)).Count(&result.Total).Error; err != nil {
		return result, stderr.Wrap(stderr.CodeDatabaseError, err, "pagination: count failed")
	}
	result.TotalPages = int((result.Total + int64(size) - 1) / int64(size))

	offset := (int64(page) - 1) * int64(size)
	if offset >= result.Total {
		return result, nil
	}

	var items []T
	if err := orderBy(base, q.Sorts).Offset(int(offset)).Limit(size).Find(&items).Error; err != nil {
		return result, stderr.Wrap(stderr.CodeDatabaseError, err, "pagination: page query failed")
	}
	result.Items = items
	return result, nil
}
