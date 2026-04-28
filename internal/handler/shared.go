package handler

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// --- numeric ---

func numericFromString(s string) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(s)
	return n
}

func nullNumericFromPtr(s *string) pgtype.Numeric {
	if s == nil {
		return pgtype.Numeric{}
	}
	return numericFromString(*s)
}

func numericToString(n pgtype.Numeric) string {
	if !n.Valid {
		return "0"
	}
	f, _ := n.Float64Value()
	return strconv.FormatFloat(f.Float64, 'f', 8, 64)
}

func nullNumericToPtr(n pgtype.Numeric) *string {
	if !n.Valid {
		return nil
	}
	s := numericToString(n)
	return &s
}

func numericToFloat(n pgtype.Numeric) float64 {
	f, _ := n.Float64Value()
	if f.Float64 == 0 {
		return 0
	}
	return f.Float64
}

// --- text ---

func nullTextFromPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func nullTextToPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func strDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// --- uuid ---

func uuidFromString(s string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// nullNumericSubtract returns a - b as pgtype.Numeric. If a is nil, returns zero.
func nullNumericSubtract(a, b *string) pgtype.Numeric {
	if a == nil || *a == "" {
		return pgtype.Numeric{}
	}
	aVal, _ := new(big.Float).SetString(*a)
	if b != nil && *b != "" && *b != "0" {
		bVal, _ := new(big.Float).SetString(*b)
		aVal.Sub(aVal, bVal)
	}
	result, _ := aVal.Float64()
	s := strconv.FormatFloat(result, 'f', -1, 64)
	return numericFromString(s)
}

func commissionDesc(transferDesc *string) pgtype.Text {
	if transferDesc != nil && *transferDesc != "" {
		return pgtype.Text{String: "Комиссия · " + *transferDesc, Valid: true}
	}
	return pgtype.Text{String: "Комиссия", Valid: true}
}

func nullUUIDFromPtr(s *string) pgtype.UUID {
	if s == nil {
		return pgtype.UUID{}
	}
	return uuidFromString(*s)
}

func nullUUIDToPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuidToString(u)
	return &s
}

// --- date ---

func dateFromString(s string) pgtype.Date {
	t, _ := time.Parse("2006-01-02", s)
	return pgtype.Date{Time: t, Valid: true}
}

func nullDateFromPtr(s *string) pgtype.Date {
	if s == nil {
		return pgtype.Date{}
	}
	return dateFromString(*s)
}

// --- system categories (fixed UUIDs from schema seed) ---

func systemCategoryUUID(name string) pgtype.UUID {
	ids := map[string]string{
		"bank-fees": "00000000-0000-0000-0000-000000000001",
		"exchange":  "00000000-0000-0000-0000-000000000002",
		"food":      "00000000-0000-0000-0000-000000000003",
		"transport": "00000000-0000-0000-0000-000000000004",
		"income":    "00000000-0000-0000-0000-000000000005",
	}
	return uuidFromString(ids[name])
}
