// Package pagination provides keyset (cursor) and offset pagination for GORM queries.
//
// Cursor pagination builds a lexicographic keyset predicate over any number of sort columns
// with mixed directions, and returns an HMAC-signed opaque token so clients cannot tamper
// with the position. Offset pagination returns totals for classic page navigation.
package pagination
