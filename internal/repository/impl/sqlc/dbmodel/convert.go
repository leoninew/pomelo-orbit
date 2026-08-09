package dbmodel

import (
	"database/sql"
	"time"
)

func NullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func StringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func NullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func TimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func BoolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func IntBool(value int64) bool {
	return value != 0
}

func NullInt64FromBoolPtr(value *bool) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: BoolInt(*value), Valid: true}
}

func BoolPtrFromNullInt64(value sql.NullInt64) *bool {
	if !value.Valid {
		return nil
	}
	result := IntBool(value.Int64)
	return &result
}

func NullInt64FromIntPtr(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

func IntPtrFromNullInt64(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	v := int(value.Int64)
	return &v
}
