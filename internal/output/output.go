// Package output renders command results as table, JSON or YAML.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

type Format string

const (
	Auto  Format = "auto"
	JSON  Format = "json"
	YAML  Format = "yaml"
	Table Format = "table"
)

// Tabular values can render as a table; others fall back to YAML in table mode.
type Tabular interface {
	Table() (header []string, rows [][]string)
}

func ParseFormat(s string) (Format, error) {
	switch f := Format(s); f {
	case Auto, JSON, YAML, Table:
		return f, nil
	}
	return "", fmt.Errorf("invalid output format %q: want auto, json, yaml or table", s)
}

// Resolve maps Auto to Table on a terminal and JSON otherwise.
func Resolve(f Format, tty bool) Format {
	if f != Auto {
		return f
	}
	if tty {
		return Table
	}
	return JSON
}

func IsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// Write renders v in format f. f must already be resolved.
func Write(w io.Writer, f Format, v any) error {
	switch f {
	case JSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case Table:
		if t, ok := v.(Tabular); ok {
			return writeTable(w, t)
		}
		return writeYAML(w, v)
	case YAML:
		return writeYAML(w, v)
	}
	return fmt.Errorf("unresolved output format %q", f)
}

func writeYAML(w io.Writer, v any) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return enc.Close()
}

func writeTable(w io.Writer, t Tabular) error {
	header, rows := t.Table()
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	return tw.Flush()
}
