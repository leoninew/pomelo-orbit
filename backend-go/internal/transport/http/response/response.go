package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"backend/internal/repository"
)

type PaginatedResp[T any] struct {
	Items   []T `json:"items"`
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Pages   int `json:"pages"`
}

func NewPaginatedResp[T any](page repository.Page[T]) PaginatedResp[T] {
	pages := 0
	if page.PerPage > 0 {
		pages = (page.Total + page.PerPage - 1) / page.PerPage
	}
	if page.Items == nil {
		page.Items = []T{}
	}
	return PaginatedResp[T]{Items: page.Items, Total: page.Total, Page: page.Page, PerPage: page.PerPage, Pages: pages}
}

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
		slog.Error("write response failed", "error", err)
	}
}

func QueryInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func QueryProjectId(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func FormatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func FormatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := FormatTime(*value)
	return &formatted
}
