# Package `pagination`

Keyset (cursor) and offset pagination for GORM queries.

## Cursor pagination

Correct for any number of sort columns with mixed `ASC`/`DESC`, and tamper-proof:

```go
codec, err := pagination.NewCodec(cursorKey) // >= 32 bytes, from your config/secret store

sorts := []pagination.Sort{{Column: "created_at", Desc: true}, {Column: "id", Desc: true}}
page, err := pagination.Cursor(ctx, db.Model(&Post{}).Where("author_id = ?", id), codec,
    pagination.CursorQuery[Post]{
        Sorts: sorts,
        Key:   func(p Post) []any { return []any{p.CreatedAt, p.ID} }, // same order as Sorts
    },
    pagination.CursorRequest{Limit: req.Limit, Cursor: req.Cursor}, // bind from query string
)
// page.Items, page.HasMore, page.NextCursor (pass back as ?cursor=...)
```

* The last sort must be unique (primary key) so the order is total.
* The page condition is the lexicographic keyset `(a < x) OR (a = x AND b < y) ...`; it never skips or repeats rows, unlike per-column `AND` filters.
* `NextCursor` is built from the **last returned row**, not from the extra look-ahead row.
* The token is `base64url(payload).base64url(HMAC-SHA256)`, with typed values (int64, time with nanoseconds) and a sort fingerprint. A tampered token, another key, another sort or an unknown version returns `stderr.CodeInvalidParam` with key `pagination.invalid_cursor`.
* Sort values must be non-NULL (`COALESCE` nullable columns). Sort columns are validated (`name` or `table.name`) and quoted by GORM; values are always bound parameters.
* `Limit <= 0` uses `DefaultLimit` (20); the limit is capped at `MaxLimit` (100).

## Offset pagination

```go
page, err := pagination.Offset[Post](ctx, db.Where("author_id = ?", id),
    pagination.OffsetQuery{Sorts: []pagination.Sort{{Column: "id"}}},
    pagination.OffsetRequest{Page: req.Page, PageSize: req.PageSize},
)
// page.Items, page.Page, page.PageSize, page.Total, page.TotalPages
```

The total is `COUNT(*)` over the query you pass in, so avoid `GROUP BY` there (use a subquery). `Page < 1` becomes 1; a page beyond the end returns empty `Items` with correct totals.

Prefer cursor pagination for large or fast-changing tables; offset cost grows with the page number.
