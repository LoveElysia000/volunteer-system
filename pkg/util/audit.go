package util

import (
	"encoding/json"
)

const (
	DefaultPage     int32 = 1
	DefaultPageSize int32 = 20
	MaxPageSize     int32 = 100
)

// NormalizePagination normalizes page and pageSize and returns pageSize/offset for db queries.
func NormalizePagination(page, pageSize int32) (limit int, offset int, normalizedPage int32, normalizedPageSize int32) {
	if page <= 0 {
		page = DefaultPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	limit = int(pageSize)
	offset = (int(page) - 1) * int(pageSize)
	return limit, offset, page, pageSize
}

// MarshalSnapshot marshals a model snapshot into a JSON string pointer for audit_records fields.
func MarshalSnapshot(v any) (string, error) {
	if v == nil {
		return "", nil
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	s := string(raw)
	return s, nil
}

// StringPtrOrNil returns nil when string is empty.
func StringPtrOrNil(s string) string {
	if s == "" {
		return ""
	}
	return s
}
