package handler

import (
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

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
	return strconv.FormatFloat(f.Float64, 'f', 2, 64)
}

func nullNumericToPtr(n pgtype.Numeric) *string {
	if !n.Valid {
		return nil
	}
	s := numericToString(n)
	return &s
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

func nullText(s *string) pgtype.Text {
	return pgtype.Text{String: strDeref(s), Valid: s != nil}
}
