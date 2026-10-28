package pagination

import (
	"context"
	"strings"

	"github.com/baby-hero-404/gocore/stderr"
	"gorm.io/gorm"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// CursorRequest is the client input for cursor pagination (query string or JSON body).
type CursorRequest struct {
	Limit  int    `form:"limit" json:"limit"`
	Cursor string `form:"cursor" json:"cursor,omitempty"`
}

// CursorPage is one page of a cursor-paginated result.
type CursorPage[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// CursorQuery configures a keyset query over T.
type CursorQuery[T any] struct {
	// Sorts is the total order of the result; the last term must be unique (e.g. the primary key).
	Sorts []Sort
	// Key returns the values of the Sorts columns for one row, in the same order. Values must be
	// non-NULL integers, floats, strings, bools or time.Time.
	Key func(T) []any
	// DefaultLimit applies when the request limit is <= 0 (default 20).
	DefaultLimit int
	// MaxLimit caps the request limit (default 100).
	MaxLimit int
}

// Cursor returns the page of T after req.Cursor. db must already carry the model/table, filters
// and joins; Cursor adds ORDER BY, the keyset predicate and the limit.
func Cursor[T any](ctx context.Context, db *gorm.DB, codec *Codec, q CursorQuery[T], req CursorRequest) (CursorPage[T], error) {
	empty := CursorPage[T]{Items: []T{}}
	if codec == nil || q.Key == nil {
		return empty, stderr.New(stderr.CodeInternal, "pagination: codec and key function are required")
	}
	if err := validateSorts(q.Sorts); err != nil {
		return empty, err
	}
	limit := clampLimit(req.Limit, q.DefaultLimit, q.MaxLimit)

	tx := orderBy(db.WithContext(ctx), q.Sorts)
	if req.Cursor != "" {
		vals, err := codec.decode(req.Cursor, q.Sorts)
		if err != nil {
			return empty, err
		}
		sql, args := keysetPredicate(tx, q.Sorts, vals)
		tx = tx.Where(sql, args...)
	}

	var items []T
	if err := tx.Limit(limit + 1).Find(&items).Error; err != nil {
		return empty, stderr.Wrap(stderr.CodeDatabaseError, err, "pagination: cursor query failed")
	}

	page := CursorPage[T]{Items: items}
	if len(items) > limit {
		page.HasMore = true
		page.Items = items[:limit]
		vals := q.Key(page.Items[limit-1])
		if len(vals) != len(q.Sorts) {
			return empty, stderr.New(stderr.CodeInternal, "pagination: key function must return one value per sort")
		}
		token, err := codec.encode(q.Sorts, vals)
		if err != nil {
			return empty, err
		}
		page.NextCursor = token
	}
	if page.Items == nil {
		page.Items = []T{}
	}
	return page, nil
}

// keysetPredicate builds the lexicographic "rows after this position" condition:
//
//	(c0 op v0) OR (c0 = v0 AND c1 op v1) OR ... OR (c0 = v0 AND ... AND cn op vn)
//
// where op is ">" for ASC and "<" for DESC, so mixed directions work.
func keysetPredicate(db *gorm.DB, sorts []Sort, vals []any) (string, []any) {
	cols := make([]string, len(sorts))
	for i, s := range sorts {
		cols[i] = db.Statement.Quote(s.clauseColumn())
	}
	var groups []string
	var args []any
	for i, s := range sorts {
		var terms []string
		for j := 0; j < i; j++ {
			terms = append(terms, cols[j]+" = ?")
			args = append(args, vals[j])
		}
		op := ">"
		if s.Desc {
			op = "<"
		}
		terms = append(terms, cols[i]+" "+op+" ?")
		args = append(args, vals[i])
		groups = append(groups, "("+strings.Join(terms, " AND ")+")")
	}
	return "(" + strings.Join(groups, " OR ") + ")", args
}

func clampLimit(limit, def, max int) int {
	if def <= 0 {
		def = defaultLimit
	}
	if max <= 0 {
		max = maxLimit
	}
	if limit <= 0 {
		limit = def
	}
	return min(limit, max)
}
