package pagination

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/baby-hero-404/gocore/stderr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Sort is one ORDER BY term. For cursor pagination the last term must uniquely identify a
// row (typically the primary key) so the order is total.
type Sort struct {
	Column string // "name" or "table.name"; letters, digits and underscores only
	Desc   bool
}

var columnPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

func validateSorts(sorts []Sort) error {
	if len(sorts) == 0 {
		return stderr.New(stderr.CodeInternal, "pagination: at least one sort is required")
	}
	for _, s := range sorts {
		if !columnPattern.MatchString(s.Column) {
			return stderr.Newf(stderr.CodeInternal, "pagination: invalid sort column %q", s.Column)
		}
	}
	return nil
}

func (s Sort) clauseColumn() clause.Column {
	if table, name, ok := strings.Cut(s.Column, "."); ok {
		return clause.Column{Table: table, Name: name}
	}
	return clause.Column{Name: s.Column}
}

func orderBy(db *gorm.DB, sorts []Sort) *gorm.DB {
	cols := make([]clause.OrderByColumn, len(sorts))
	for i, s := range sorts {
		cols[i] = clause.OrderByColumn{Column: s.clauseColumn(), Desc: s.Desc}
	}
	return db.Order(clause.OrderBy{Columns: cols})
}

// fingerprint identifies a sort list so a cursor issued for one ordering is rejected by another.
func fingerprint(sorts []Sort) string {
	h := sha256.New()
	for _, s := range sorts {
		h.Write([]byte(s.Column))
		if s.Desc {
			h.Write([]byte{1})
		} else {
			h.Write([]byte{0})
		}
		h.Write([]byte{0xff})
	}
	return hex.EncodeToString(h.Sum(nil)[:8])
}
