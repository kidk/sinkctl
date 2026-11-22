package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const sample = `current-context: prod
contexts:
  prod:
    url: https://prod.example
    token: prod-token
  dev:
    url: https://dev.example
    token: dev-token
`

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadMissingFile(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.CurrentContext != "" || len(c.Contexts) != 0 {
		t.Fatalf("want empty config, got %+v", c)
	}
}

func TestResolve(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(sample), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		ctx     string
		env     map[string]string
		want    Context
		wantErr bool
	}{
		{"current context", "", nil, Context{"https://prod.example", "prod-token"}, false},
		{"named context", "dev", nil, Context{"https://dev.example", "dev-token"}, false},
		{"env overrides file", "dev", map[string]string{EnvToken: "env-token"}, Context{"https://dev.example", "env-token"}, false},
		{"unknown context", "staging", nil, Context{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := c.Resolve(tc.ctx, env(tc.env))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestTokenRedacted(t *testing.T) {
	c := Context{URL: "https://x", Token: "secret"}
	j, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	y, err := yaml.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, out := range []string{string(j), string(y)} {
		if strings.Contains(out, "secret") || !strings.Contains(out, "***") {
			t.Errorf("token not redacted: %s", out)
		}
	}
}

func TestResolveEnvOnly(t *testing.T) {
	got, err := (&Config{}).Resolve("", env(map[string]string{EnvURL: "https://x", EnvToken: "t"}))
	if err != nil {
		t.Fatal(err)
	}
	if got != (Context{"https://x", "t"}) {
		t.Errorf("got %+v", got)
	}
	if _, err := (&Config{}).Resolve("", env(nil)); err == nil {
		t.Error("want error with no config and no env")
	}
}
