package app

import "strings"

const (
	defaultListPage     = 1
	defaultListPageSize = 20
	maxListPageSize     = 100
	defaultSortField    = "created_at"
	defaultSortDir      = "asc"
)

type ListQuery struct {
	Page      int
	PageSize  int
	Search    string
	SortField string
	SortDir   string
}

type ListResult[T any] struct {
	Items    []T `json:"items" toon:"items"`
	Total    int `json:"total" toon:"total"`
	Page     int `json:"page" toon:"page"`
	PageSize int `json:"page_size" toon:"page_size"`
}

func (q ListQuery) Normalize() ListQuery {
	q.Page = normalizeListPage(q.Page)
	q.PageSize = normalizeListPageSize(q.PageSize)
	q.Search = strings.TrimSpace(q.Search)
	q.SortField = normalizeListSortField(q.SortField)
	q.SortDir = normalizeListSortDir(q.SortDir)
	return q
}

func NewListResult[T any](query ListQuery, items []T, total int) ListResult[T] {
	query = query.Normalize()
	return ListResult[T]{
		Items:    items,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}
}

func normalizeListPage(value int) int {
	if value <= 0 {
		return defaultListPage
	}
	return value
}

func normalizeListPageSize(value int) int {
	if value <= 0 {
		return defaultListPageSize
	}
	if value > maxListPageSize {
		return maxListPageSize
	}
	return value
}

func normalizeListSortField(value string) string {
	field := strings.ToLower(strings.TrimSpace(value))
	if field == "" {
		return defaultSortField
	}
	if field == "name" {
		return "legal_name"
	}
	return field
}

func normalizeListSortDir(value string) string {
	dir := strings.ToLower(strings.TrimSpace(value))
	switch dir {
	case "", "asc":
		return defaultSortDir
	case "desc":
		return dir
	default:
		return defaultSortDir
	}
}
