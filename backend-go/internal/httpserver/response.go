package httpserver

import (
	"strconv"
	"time"

	"backend/internal/orbit"
)

type paginatedResp[T any] struct {
	Items   []T `json:"items"`
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Pages   int `json:"pages"`
}

func newPaginatedResp[T any](page orbit.Page[T]) paginatedResp[T] {
	pages := 0
	if page.PerPage > 0 {
		pages = (page.Total + page.PerPage - 1) / page.PerPage
	}
	if page.Items == nil {
		page.Items = []T{}
	}
	return paginatedResp[T]{Items: page.Items, Total: page.Total, Page: page.Page, PerPage: page.PerPage, Pages: pages}
}

func queryInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func queryProjectId(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := formatTime(*value)
	return &formatted
}
