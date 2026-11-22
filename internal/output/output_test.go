package output

import (
	"bytes"
	"testing"
)

type row struct {
	Name string `json:"name" yaml:"name"`
}

func (r row) Table() ([]string, [][]string) {
	return []string{"NAME"}, [][]string{{r.Name}}
}

type plain struct {
	Name string `json:"name" yaml:"name"`
}

func TestResolve(t *testing.T) {
	cases := []struct {
		in   Format
		tty  bool
		want Format
	}{
		{Auto, true, Table},
		{Auto, false, JSON},
		{YAML, true, YAML},
		{Table, false, Table},
	}
	for _, c := range cases {
		if got := Resolve(c.in, c.tty); got != c.want {
			t.Errorf("Resolve(%q, %v) = %q, want %q", c.in, c.tty, got, c.want)
		}
	}
}

func TestParseFormat(t *testing.T) {
	if _, err := ParseFormat("xml"); err == nil {
		t.Fatal("want error for xml")
	}
	if f, err := ParseFormat("json"); err != nil || f != JSON {
		t.Fatalf("got %q, %v", f, err)
	}
}

func TestWrite(t *testing.T) {
	cases := []struct {
		name string
		f    Format
		v    any
		want string
	}{
		{"json", JSON, row{"a"}, "{\n  \"name\": \"a\"\n}\n"},
		{"yaml", YAML, row{"a"}, "name: a\n"},
		{"table", Table, row{"a"}, "NAME\na\n"},
		{"table fallback", Table, plain{"a"}, "name: a\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := Write(&buf, c.f, c.v); err != nil {
				t.Fatal(err)
			}
			if buf.String() != c.want {
				t.Errorf("got %q, want %q", buf.String(), c.want)
			}
		})
	}
}
