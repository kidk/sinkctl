package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// buffers are non-TTY, so auto output resolves to JSON
func exec(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	return execAPI(t, "", args...)
}

// execAPI runs against Bugsink at url; empty url means no credentials.
func execAPI(t *testing.T, url string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	token := ""
	if url != "" {
		token = "tok"
	}
	t.Setenv("BUGSINK_URL", url)
	t.Setenv("BUGSINK_TOKEN", token)
	return execIn(t, t.TempDir(), "", args...)
}

// execIn runs with config under xdg and the given stdin.
func execIn(t *testing.T, xdg, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	var out, errb bytes.Buffer
	code = Run(t.Context(), args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func TestVersionJSON(t *testing.T) {
	code, out, _ := exec(t, "version")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if got["version"] != "dev" {
		t.Errorf("version = %q", got["version"])
	}
}

func TestExitCodes(t *testing.T) {
	cases := []struct {
		name string
		args []string
		code int
		kind string // empty: plain-text error, since -o itself is invalid
	}{
		{"invalid output", []string{"version", "-o", "xml"}, 2, ""},
		{"unknown flag", []string{"version", "--nope"}, 2, "usage"},
		{"extra args", []string{"version", "x"}, 2, "usage"},
		{"unknown command", []string{"nope"}, 2, "usage"},
		{"mcp without creds", []string{"mcp"}, 3, "auth"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, _, stderr := exec(t, c.args...)
			if code != c.code {
				t.Errorf("exit %d, want %d (stderr %q)", code, c.code, stderr)
			}
			if c.kind == "" {
				return
			}
			var env struct {
				Error struct{ Kind string } `json:"error"`
			}
			if err := json.Unmarshal([]byte(stderr), &env); err != nil {
				t.Fatalf("stderr not JSON: %q", stderr)
			}
			if env.Error.Kind != c.kind {
				t.Errorf("kind %q, want %q", env.Error.Kind, c.kind)
			}
		})
	}
}

func TestNoCredsLoginHint(t *testing.T) {
	code, _, stderr := exec(t, "projects", "list")
	if code != 3 || !strings.Contains(stderr, "sinkctl config login") {
		t.Errorf("exit %d, stderr %q", code, stderr)
	}
}

func TestPlainErrorWithYAML(t *testing.T) {
	_, _, stderr := exec(t, "mcp", "-o", "yaml")
	if stderr == "" || stderr[0] == '{' {
		t.Errorf("want plain text error, got %q", stderr)
	}
}
