package ops

import (
	"fmt"
	"strconv"
	"time"

	"github.com/kidk/sinkctl/internal/bugsink"
)

type EventCounts struct{ bugsink.EventCounts }

func (c EventCounts) Table() ([]string, [][]string) {
	rows := make([][]string, 0, len(c.Buckets))
	for _, b := range c.Buckets {
		rows = append(rows, []string{b.Timestamp.Format(time.RFC3339), strconv.Itoa(b.Count)})
	}
	return []string{"HOUR", "COUNT"}, rows
}

func parseRange(start, end string) (s, e *time.Time, err error) {
	if s, err = parseTime("start", start); err != nil {
		return nil, nil, err
	}
	e, err = parseTime("end", end)
	return s, e, err
}

func parseTime(field, v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, usagef("%s: want RFC 3339 time like 2026-01-02T15:00:00Z, got %q", field, v)
	}
	return &t, nil
}

func fmtPtr[T any](p *T) string {
	if p == nil {
		return ""
	}
	return fmt.Sprint(*p)
}
