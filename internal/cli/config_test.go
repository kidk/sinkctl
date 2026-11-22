package cli

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kidk/sinkctl/internal/config"
)

// loginAPI accepts only token "good".
func loginAPI(t *testing.T) string {
	return fakeAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, `{"results":[]}`)
	})
}

func loadIn(t *testing.T, xdg string) *config.Config {
	t.Helper()
	c, err := config.Load(filepath.Join(xdg, "sinkctl", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func noEnv(t *testing.T) {
	t.Setenv("BUGSINK_URL", "")
	t.Setenv("BUGSINK_TOKEN", "")
}

func TestLoginThenUse(t *testing.T) {
	noEnv(t)
	xdg, url := t.TempDir(), loginAPI(t)

	code, out, stderr := execIn(t, xdg, "good\n", "config", "login", "prod", "--url", url+"/", "--token-stdin")
	if code != 0 || !strings.Contains(out, `"context": "prod"`) {
		t.Fatalf("exit %d: %s %s", code, out, stderr)
	}
	if strings.Contains(out, "good") {
		t.Error("token printed")
	}
	if code, _, stderr = execIn(t, xdg, "x\n", "config", "login", "dev", "--url", "http://127.0.0.1:1", "--token-stdin", "--no-verify"); code != 0 {
		t.Fatalf("no-verify login: exit %d: %s", code, stderr)
	}
	if c := loadIn(t, xdg); c.CurrentContext != "dev" || c.Contexts["prod"] != (config.Context{URL: url, Token: "good"}) {
		t.Fatalf("config = %+v", c)
	}

	if code, _, stderr = execIn(t, xdg, "", "config", "use", "prod"); code != 0 {
		t.Fatalf("use: exit %d: %s", code, stderr)
	}
	if code, _, stderr = execIn(t, xdg, "", "projects", "list"); code != 0 {
		t.Errorf("projects list via saved context: exit %d: %s", code, stderr)
	}
}

func TestLoginBadTokenWritesNothing(t *testing.T) {
	noEnv(t)
	xdg := t.TempDir()
	code, _, _ := execIn(t, xdg, "bad\n", "config", "login", "prod", "--url", loginAPI(t), "--token-stdin")
	if code != 3 {
		t.Errorf("exit %d, want 3", code)
	}
	if _, err := os.Stat(filepath.Join(xdg, "sinkctl")); !os.IsNotExist(err) {
		t.Errorf("config dir created: %v", err)
	}
}

func TestConfigErrors(t *testing.T) {
	noEnv(t)
	cases := []struct {
		args []string
		code int
	}{
		{[]string{"config", "login", "prod", "--url", "http://x"}, 2}, // non-TTY stdin, no --token-stdin
		{[]string{"config", "login", "prod", "--token-stdin"}, 2},
		{[]string{"config", "use", "nope"}, 3},
	}
	for _, c := range cases {
		if code, _, stderr := execIn(t, t.TempDir(), "tok\n", c.args...); code != c.code {
			t.Errorf("%v: exit %d, want %d: %s", c.args, code, c.code, stderr)
		}
	}
}
