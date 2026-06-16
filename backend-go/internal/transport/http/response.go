package transporthttp

import (
	"net/http"
	"time"

	"backend/internal/repository"
	transportresponse "backend/internal/transport/http/response"
)

type paginatedResp[T any] = transportresponse.PaginatedResp[T]

func newPaginatedResp[T any](page repository.Page[T]) paginatedResp[T] {
	return transportresponse.NewPaginatedResp(page)
}

func queryInt(value string, fallback int) int {
	return transportresponse.QueryInt(value, fallback)
}

func queryProjectId(value string) *string {
	return transportresponse.QueryProjectId(value)
}

func formatTime(value time.Time) string {
	return transportresponse.FormatTime(value)
}

func formatOptionalTime(value *time.Time) *string {
	return transportresponse.FormatOptionalTime(value)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	transportresponse.JSON(w, status, value)
}
