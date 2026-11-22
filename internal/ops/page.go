package ops

import (
	"context"
	"fmt"
	"net/url"
)

// Page is one page of a cursor-paginated list.
type Page[T any] struct {
	Results    []T    `json:"results"`
	NextCursor string `json:"next_cursor,omitempty"`

	header []string
	row    func(T) []string
}

func (p Page[T]) Table() ([]string, [][]string) {
	rows := make([][]string, 0, len(p.Results))
	for _, r := range p.Results {
		rows = append(rows, p.row(r))
	}
	return p.header, rows
}

func newPage[T any](results []T, next *string, header []string, row func(T) []string) (Page[T], error) {
	c, err := cursorOf(next)
	return Page[T]{Results: results, NextCursor: c, header: header, row: row}, err
}

// cursorOf extracts the cursor query param from the API's next URL; errors so --all never returns a silent partial list.
func cursorOf(next *string) (string, error) {
	if next == nil {
		return "", nil
	}
	u, err := url.Parse(*next)
	if err == nil && u.Query().Get("cursor") != "" {
		return u.Query().Get("cursor"), nil
	}
	return "", &Error{Kind: KindAPI, Message: fmt.Sprintf("next page URL has no cursor: %q", *next)}
}

// All follows next cursors from cursor until the last page.
func All[T any](ctx context.Context, cursor string, fetch func(ctx context.Context, cursor string) (Page[T], error)) (Page[T], error) {
	all := Page[T]{Results: []T{}}
	for {
		p, err := fetch(ctx, cursor)
		if err != nil {
			return all, err
		}
		all.Results = append(all.Results, p.Results...)
		all.header, all.row = p.header, p.row
		if p.NextCursor == "" {
			return all, nil
		}
		cursor = p.NextCursor
	}
}
