package repository

type Page[T any] struct {
	Items   []T
	Total   int
	Page    int
	PerPage int
}

func NormalizePage(page int, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}
