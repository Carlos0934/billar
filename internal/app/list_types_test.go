package app

import (
	"reflect"
	"testing"
)

func TestListQueryNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query ListQuery
		want  ListQuery
	}{
		{
			name:  "defaults",
			query: ListQuery{},
			want:  ListQuery{Page: 1, PageSize: 20, SortField: "created_at", SortDir: "asc"},
		},
		{
			name: "clamps and normalizes aliases",
			query: ListQuery{
				Page:      -2,
				PageSize:  200,
				Search:    "  Acme  ",
				SortField: " name ",
				SortDir:   " DESC ",
			},
			want: ListQuery{Page: 1, PageSize: 100, Search: "Acme", SortField: "legal_name", SortDir: "desc"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.query.Normalize()
			if got != tc.want {
				t.Fatalf("Normalize() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestNewListResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query ListQuery
		items []string
		total int
		want  ListResult[string]
	}{
		{
			name:  "populates fields",
			query: ListQuery{Page: 2, PageSize: 10}.Normalize(),
			items: []string{"a", "b"},
			total: 12,
			want:  ListResult[string]{Items: []string{"a", "b"}, Total: 12, Page: 2, PageSize: 10},
		},
		{
			name:  "preserves empty page slice",
			query: ListQuery{Page: 1, PageSize: 20}.Normalize(),
			items: []string{},
			total: 0,
			want:  ListResult[string]{Items: []string{}, Total: 0, Page: 1, PageSize: 20},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := NewListResult(tc.query, tc.items, tc.total)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("NewListResult() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
