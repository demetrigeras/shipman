package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

func nullableUUID(id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	return *id
}

func uuidPtrNullable(ns sql.NullString) *uuid.UUID {
	if !ns.Valid {
		return nil
	}
	val, err := uuid.Parse(ns.String)
	if err != nil {
		return nil
	}
	return &val
}

func nullableString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func nullableFloat(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}

func nullableBytes(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

func nullableInt16(i *int16) any {
	if i == nil {
		return nil
	}
	return *i
}

func stringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	v := ns.String
	return &v
}

func timePtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	t := nt.Time
	return &t
}

func floatPtr(nf sql.NullFloat64) *float64 {
	if !nf.Valid {
		return nil
	}
	v := nf.Float64
	return &v
}

func defaultString(ns sql.NullString, fallback string) string {
	if ns.Valid {
		return ns.String
	}
	return fallback
}

func bytesOrNil(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

func int16Ptr(ni sql.NullInt16) *int16 {
	if !ni.Valid {
		return nil
	}
	v := ni.Int16
	return &v
}
