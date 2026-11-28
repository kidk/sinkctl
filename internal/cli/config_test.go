package cli

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/kidk/sinkctl/internal/config"
)

// loginAPI accepts only token "good".
func loginAPI(t *testing.T) string {
	return fakeAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"detail":"Authentication credentials were not provided."}`)
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
		want string // in stderr
	}{
		{[]string{"config", "login", "prod", "--url", "http://x"}, 2, "--token-stdin"}, // non-TTY stdin
		{[]string{"config", "login", "prod", "--token-stdin"}, 2, "--url is required"},
		{[]string{"config", "login", "--url", "http://x", "--token-stdin"}, 2, "name is required"},
		{[]string{"config", "login"}, 2, "--url is required"},
		{[]string{"config", "use", "nope"}, 3, "not found"},
	}
	for _, c := range cases {
		code, _, stderr := execIn(t, t.TempDir(), "tok\n", c.args...)
		if code != c.code || !strings.Contains(stderr, c.want) {
			t.Errorf("%v: exit %d, want %d with %q: %s", c.args, code, c.code, c.want, stderr)
		}
	}
}

// fakeTTY makes login prompt as on a terminal; the hidden token prompt reads the next stdin line.
func fakeTTY(t *testing.T) {
	orig := newPrompter
	t.Cleanup(func() { newPrompter = orig })
	newPrompter = func(cmd *cobra.Command) *prompter {
		p := orig(cmd)
		p.tty = true
		p.password = func() (string, error) { return p.r.ReadString('\n') }
		return p
	}
}

func TestGuidedLogin(t *testing.T) {
	noEnv(t)
	fakeTTY(t)
	xdg, url := t.TempDir(), loginAPI(t)

	// Enter at the name prompt takes the URL host.
	code, out, stderr := execIn(t, xdg, url+"/\n\ngood\n", "config", "login")
	if code != 0 || !strings.Contains(stderr, "Context name [127.0.0.1]: ") {
		t.Fatalf("exit %d: %s %s", code, out, stderr)
	}
	if c := loadIn(t, xdg); c.CurrentContext != "127.0.0.1" || c.Contexts["127.0.0.1"] != (config.Context{URL: url, Token: "good"}) {
		t.Fatalf("config = %+v", c)
	}
}

func TestGuidedLoginPromptsOnlyMissing(t *testing.T) {
	noEnv(t)
	fakeTTY(t)
	xdg := t.TempDir()
	code, _, stderr := execIn(t, xdg, "good\n", "config", "login", "prod", "--url", loginAPI(t))
	if code != 0 || strings.Contains(stderr, "URL:") || strings.Contains(stderr, "Context name") || !strings.Contains(stderr, "Token: ") {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if c := loadIn(t, xdg); c.CurrentContext != "prod" {
		t.Fatalf("config = %+v", c)
	}
}

func TestGuidedLoginRetries(t *testing.T) {
	noEnv(t)
	fakeTTY(t)
	xdg, url := t.TempDir(), loginAPI(t)
	notBugsink := fakeAPI(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "<html>nope</html>")
	})

	stdin := "http://127.0.0.1:1\n" + notBugsink + "\n" + url + "\nprod\nbad\ngood\n"
	code, _, stderr := execIn(t, xdg, stdin, "config", "login")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	for _, want := range []string{"cannot reach http://127.0.0.1:1", "does not look like Bugsink", "401 Unauthorized"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr lacks %q: %s", want, stderr)
		}
	}
	if c := loadIn(t, xdg); c.Contexts["prod"].Token != "good" {
		t.Fatalf("config = %+v", c)
	}
}

func TestGuidedLoginGivesUpAfterThreeTokens(t *testing.T) {
	noEnv(t)
	fakeTTY(t)
	xdg := t.TempDir()
	code, _, stderr := execIn(t, xdg, loginAPI(t)+"\nprod\nbad\nbad\nbad\ngood\n", "config", "login")
	if code != 3 {
		t.Errorf("exit %d, want 3: %s", code, stderr)
	}
	if n := strings.Count(stderr, "Token: "); n != 3 {
		t.Errorf("%d token prompts, want 3", n)
	}
	if _, err := os.Stat(filepath.Join(xdg, "sinkctl")); !os.IsNotExist(err) {
		t.Errorf("config dir created: %v", err)
	}
}
