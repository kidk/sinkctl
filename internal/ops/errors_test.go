package ops

import (
	"errors"
	"testing"
)

func TestHTTPError(t *testing.T) {
	if err := HTTPError(200, nil); err != nil {
		t.Fatalf("2xx: got %v", err)
	}
	cases := []struct {
		status int
		want   Kind
	}{
		{401, KindAuth},
		{403, KindAuth},
		{404, KindNotFound},
		{400, KindAPI},
		{500, KindAPI},
	}
	for _, c := range cases {
		var e *Error
		if !errors.As(HTTPError(c.status, []byte(`{"detail":"x"}`)), &e) {
			t.Fatalf("%d: not *Error", c.status)
		}
		if e.Kind != c.want || e.Status != c.status {
			t.Errorf("%d: got kind %q status %d", c.status, e.Kind, e.Status)
		}
	}
}
