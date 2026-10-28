package pagination

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var dbSeq atomic.Int64

type row struct {
	ID int64 `gorm:"primaryKey"`
	A  int
	B  int
	N  string
	At time.Time
}

var testKey = []byte("0123456789abcdef0123456789abcdef")

func newDB(t *testing.T, rows int) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:pagination%d?mode=memory&cache=shared", dbSeq.Add(1))), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&row{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 123456789, time.UTC)
	for i := 1; i <= rows; i++ {
		r := row{ID: int64(i), A: i % 5, B: i % 7, N: fmt.Sprintf("n%02d", i%11), At: base.Add(time.Duration(i%13) * time.Hour)}
		if err := db.Create(&r).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return db
}

func keyOf(sorts []Sort) func(row) []any {
	return func(r row) []any {
		out := make([]any, len(sorts))
		for i, s := range sorts {
			switch s.Column {
			case "a":
				out[i] = r.A
			case "b":
				out[i] = r.B
			case "n":
				out[i] = r.N
			case "at":
				out[i] = r.At
			default:
				out[i] = r.ID
			}
		}
		return out
	}
}

func orderSQL(sorts []Sort) string {
	parts := make([]string, len(sorts))
	for i, s := range sorts {
		parts[i] = s.Column
		if s.Desc {
			parts[i] += " DESC"
		}
	}
	return strings.Join(parts, ", ")
}

func TestCursor_WalksAllPagesWithoutGapsOrDuplicates(t *testing.T) {
	db := newDB(t, 200)
	codec, err := NewCodec(testKey)
	if err != nil {
		t.Fatal(err)
	}
	id := Sort{Column: "id"}
	idDesc := Sort{Column: "id", Desc: true}

	tests := []struct {
		name  string
		sorts []Sort
		limit int
	}{
		{"one column asc", []Sort{id}, 7},
		{"one column desc", []Sort{idDesc}, 7},
		{"two columns asc", []Sort{{Column: "a"}, id}, 7},
		{"two columns desc", []Sort{{Column: "a", Desc: true}, idDesc}, 7},
		{"two columns mixed", []Sort{{Column: "a"}, idDesc}, 9},
		{"three columns mixed", []Sort{{Column: "a", Desc: true}, {Column: "b"}, id}, 11},
		{"string and time columns", []Sort{{Column: "n"}, {Column: "at", Desc: true}, id}, 13},
		{"limit one", []Sort{{Column: "a"}, {Column: "b", Desc: true}, id}, 1},
		{"limit larger than data", []Sort{{Column: "a"}, id}, 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var want []row
			if err := db.Order(orderSQL(tt.sorts)).Find(&want).Error; err != nil {
				t.Fatal(err)
			}
			q := CursorQuery[row]{Sorts: tt.sorts, Key: keyOf(tt.sorts), MaxLimit: 1000}
			var got []row
			cursor := ""
			for pages := 0; pages < 1000; pages++ {
				page, err := Cursor(context.Background(), db.Model(&row{}), codec, q, CursorRequest{Limit: tt.limit, Cursor: cursor})
				if err != nil {
					t.Fatalf("page %d: %v", pages, err)
				}
				got = append(got, page.Items...)
				if !page.HasMore {
					if page.NextCursor != "" {
						t.Fatal("last page must not carry a cursor")
					}
					break
				}
				cursor = page.NextCursor
			}
			if len(got) != len(want) {
				t.Fatalf("got %d rows, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i].ID != want[i].ID {
					t.Fatalf("row %d: got id %d, want %d", i, got[i].ID, want[i].ID)
				}
			}
		})
	}
}

func TestCursor_LimitHandling(t *testing.T) {
	db := newDB(t, 30)
	codec, _ := NewCodec(testKey)
	sorts := []Sort{{Column: "id"}}
	tests := []struct {
		name         string
		q            CursorQuery[row]
		limit        int
		wantLen      int
		wantHasMore  bool
		wantNonEmpty bool
	}{
		{"default limit", CursorQuery[row]{Sorts: sorts, Key: keyOf(sorts), DefaultLimit: 5}, 0, 5, true, true},
		{"negative uses default", CursorQuery[row]{Sorts: sorts, Key: keyOf(sorts), DefaultLimit: 4}, -3, 4, true, true},
		{"capped at max", CursorQuery[row]{Sorts: sorts, Key: keyOf(sorts), MaxLimit: 6}, 1000, 6, true, true},
		{"exact fit has no more", CursorQuery[row]{Sorts: sorts, Key: keyOf(sorts), MaxLimit: 100}, 30, 30, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := Cursor(context.Background(), db.Model(&row{}), codec, tt.q, CursorRequest{Limit: tt.limit})
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Items) != tt.wantLen || page.HasMore != tt.wantHasMore || (page.NextCursor != "") != tt.wantNonEmpty {
				t.Errorf("got len=%d more=%v cursor=%q", len(page.Items), page.HasMore, page.NextCursor)
			}
		})
	}
}

func TestCursor_EmptyResultIsNonNilSlice(t *testing.T) {
	db := newDB(t, 0)
	codec, _ := NewCodec(testKey)
	sorts := []Sort{{Column: "id"}}
	page, err := Cursor(context.Background(), db.Model(&row{}), codec, CursorQuery[row]{Sorts: sorts, Key: keyOf(sorts)}, CursorRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Items == nil || len(page.Items) != 0 || page.HasMore {
		t.Errorf("want empty non-nil items, got %#v", page)
	}
}

func TestCursor_RejectsBadCursors(t *testing.T) {
	db := newDB(t, 30)
	codec, _ := NewCodec(testKey)
	other, _ := NewCodec([]byte("ffffffffffffffffffffffffffffffff"))
	sorts := []Sort{{Column: "a"}, {Column: "id"}}
	q := CursorQuery[row]{Sorts: sorts, Key: keyOf(sorts), DefaultLimit: 5}
	first, err := Cursor(context.Background(), db.Model(&row{}), codec, q, CursorRequest{})
	if err != nil || first.NextCursor == "" {
		t.Fatalf("setup: %v %q", err, first.NextCursor)
	}
	good := first.NextCursor
	body, sig, _ := strings.Cut(good, ".")
	flip := func(s string) string {
		b := []byte(s)
		if b[0] == 'A' {
			b[0] = 'B'
		} else {
			b[0] = 'A'
		}
		return string(b)
	}
	otherSorts := []Sort{{Column: "a", Desc: true}, {Column: "id"}}
	otherTok, err := codec.encode(otherSorts, []any{1, int64(1)})
	if err != nil {
		t.Fatal(err)
	}
	foreign, _ := other.encode(sorts, []any{1, int64(1)})

	tests := []struct {
		name   string
		cursor string
	}{
		{"not a token", "garbage"},
		{"bad base64 body", "!!!." + sig},
		{"tampered payload", flip(body) + "." + sig},
		{"tampered signature", body + "." + flip(sig)},
		{"missing signature", body},
		{"signed with other key", foreign},
		{"issued for other sort", otherTok},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Cursor(context.Background(), db.Model(&row{}), codec, q, CursorRequest{Cursor: tt.cursor})
			var se *stderr.Error
			if !errors.As(err, &se) || se.Code != stderr.CodeInvalidParam {
				t.Fatalf("want CodeInvalidParam, got %v", err)
			}
		})
	}
}

func TestCursor_ConfigurationErrors(t *testing.T) {
	db := newDB(t, 1)
	codec, _ := NewCodec(testKey)
	key := keyOf([]Sort{{Column: "id"}})
	badKey := func(row) []any { return []any{int64(1), int64(2)} }
	db1 := newDB(t, 10)
	tests := []struct {
		name  string
		db    *gorm.DB
		codec *Codec
		q     CursorQuery[row]
	}{
		{"nil codec", db, nil, CursorQuery[row]{Sorts: []Sort{{Column: "id"}}, Key: key}},
		{"nil key", db, codec, CursorQuery[row]{Sorts: []Sort{{Column: "id"}}}},
		{"no sorts", db, codec, CursorQuery[row]{Key: key}},
		{"injection in column", db, codec, CursorQuery[row]{Sorts: []Sort{{Column: "id; DROP TABLE rows"}}, Key: key}},
		{"key arity mismatch", db1, codec, CursorQuery[row]{Sorts: []Sort{{Column: "id"}}, Key: badKey, DefaultLimit: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Cursor(context.Background(), tt.db.Model(&row{}), tt.codec, tt.q, CursorRequest{})
			var se *stderr.Error
			if !errors.As(err, &se) || se.Code != stderr.CodeInternal {
				t.Fatalf("want CodeInternal, got %v", err)
			}
		})
	}
}

func TestCursor_QueryFailureIsWrapped(t *testing.T) {
	db := newDB(t, 1)
	codec, _ := NewCodec(testKey)
	sorts := []Sort{{Column: "no_such_column"}}
	_, err := Cursor(context.Background(), db.Model(&row{}), codec, CursorQuery[row]{Sorts: sorts, Key: keyOf(sorts)}, CursorRequest{})
	var se *stderr.Error
	if !errors.As(err, &se) || se.Code != stderr.CodeDatabaseError {
		t.Fatalf("want CodeDatabaseError, got %v", err)
	}
}

func TestNewCodec_KeyLength(t *testing.T) {
	tests := []struct {
		name    string
		key     []byte
		wantErr bool
	}{
		{"nil", nil, true},
		{"short", []byte("short"), true},
		{"31 bytes", make([]byte, 31), true},
		{"32 bytes", make([]byte, 32), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCodec(tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestCodec_RoundTripKeepsTypes(t *testing.T) {
	codec, _ := NewCodec(testKey)
	sorts := []Sort{{Column: "a"}, {Column: "b"}, {Column: "c"}, {Column: "d"}, {Column: "e"}, {Column: "f"}, {Column: "g"}}
	at := time.Date(2026, 10, 28, 12, 30, 45, 123456789, time.FixedZone("x", 7*3600))
	in := []any{int64(9007199254740993), uint64(18446744073709551615), 1.5, "héllo", true, at, int8(-3)}
	token, err := codec.encode(sorts, in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := codec.decode(token, sorts)
	if err != nil {
		t.Fatal(err)
	}
	if out[0] != int64(9007199254740993) || out[1] != uint64(18446744073709551615) || out[2] != 1.5 || out[3] != "héllo" || out[4] != true || out[6] != int64(-3) {
		t.Errorf("scalar round trip mismatch: %#v", out)
	}
	if got := out[5].(time.Time); !got.Equal(at) || got.Nanosecond() != 123456789 {
		t.Errorf("time round trip mismatch: %v vs %v", got, at)
	}
}

func TestCodec_RejectsUnsupportedValues(t *testing.T) {
	codec, _ := NewCodec(testKey)
	sorts := []Sort{{Column: "a"}}
	for name, v := range map[string]any{"nil": nil, "struct": struct{}{}, "slice": []int{1}} {
		t.Run(name, func(t *testing.T) {
			if _, err := codec.encode(sorts, []any{v}); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestOffset(t *testing.T) {
	db := newDB(t, 45)
	sorts := []Sort{{Column: "id"}}
	q := OffsetQuery{Sorts: sorts, DefaultPageSize: 10, MaxPageSize: 20}
	tests := []struct {
		name       string
		req        OffsetRequest
		wantFirst  int64
		wantLen    int
		wantPage   int
		wantSize   int
		wantPages  int
		wantTotal  int64
		wantNoRows bool
	}{
		{"first page default size", OffsetRequest{}, 1, 10, 1, 10, 5, 45, false},
		{"middle page", OffsetRequest{Page: 3, PageSize: 10}, 21, 10, 3, 10, 5, 45, false},
		{"last partial page", OffsetRequest{Page: 5, PageSize: 10}, 41, 5, 5, 10, 5, 45, false},
		{"beyond last page", OffsetRequest{Page: 6, PageSize: 10}, 0, 0, 6, 10, 5, 45, true},
		{"page below one becomes one", OffsetRequest{Page: -2, PageSize: 10}, 1, 10, 1, 10, 5, 45, false},
		{"page size capped", OffsetRequest{Page: 1, PageSize: 1000}, 1, 20, 1, 20, 3, 45, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Offset[row](context.Background(), db.Model(&row{}), q, tt.req)
			if err != nil {
				t.Fatal(err)
			}
			if got.Items == nil {
				t.Fatal("items must be non-nil")
			}
			if len(got.Items) != tt.wantLen || got.Page != tt.wantPage || got.PageSize != tt.wantSize || got.TotalPages != tt.wantPages || got.Total != tt.wantTotal {
				t.Fatalf("unexpected page: len=%d page=%d size=%d pages=%d total=%d", len(got.Items), got.Page, got.PageSize, got.TotalPages, got.Total)
			}
			if !tt.wantNoRows && got.Items[0].ID != tt.wantFirst {
				t.Errorf("first id = %d, want %d", got.Items[0].ID, tt.wantFirst)
			}
		})
	}
}

func TestOffset_EmptyTableAndErrors(t *testing.T) {
	empty := newDB(t, 0)
	got, err := Offset[row](context.Background(), empty.Model(&row{}), OffsetQuery{Sorts: []Sort{{Column: "id"}}}, OffsetRequest{})
	if err != nil || got.Total != 0 || got.TotalPages != 0 || got.Items == nil || len(got.Items) != 0 {
		t.Fatalf("empty table: %+v err=%v", got, err)
	}

	db := newDB(t, 3)
	if _, err := Offset[row](context.Background(), db.Model(&row{}), OffsetQuery{}, OffsetRequest{}); err == nil {
		t.Fatal("expected error for missing sorts")
	}
	_, err = Offset[row](context.Background(), db.Model(&row{}), OffsetQuery{Sorts: []Sort{{Column: "nope"}}}, OffsetRequest{})
	var se *stderr.Error
	if !errors.As(err, &se) || se.Code != stderr.CodeDatabaseError {
		t.Fatalf("want CodeDatabaseError, got %v", err)
	}
}

func TestSort_QualifiedColumnAndValidation(t *testing.T) {
	tests := []struct {
		col string
		ok  bool
	}{
		{"id", true}, {"users.id", true}, {"_x1", true}, {"", false}, {"a b", false}, {"a;b", false}, {"a.b.c", false}, {"1a", false}, {`a"b`, false},
	}
	for _, tt := range tests {
		t.Run(tt.col, func(t *testing.T) {
			err := validateSorts([]Sort{{Column: tt.col}})
			if (err == nil) != tt.ok {
				t.Fatalf("col %q: err=%v ok=%v", tt.col, err, tt.ok)
			}
		})
	}
}
