// Package config loads Bugsink connection settings from env and the config file.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	EnvURL   = "BUGSINK_URL"
	EnvToken = "BUGSINK_TOKEN" //nolint:gosec // env var name, not a secret
)

type Context struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

type redactedContext struct {
	URL   string `json:"url" yaml:"url"`
	Token string `json:"token" yaml:"token"`
}

func (c Context) redacted() redactedContext {
	r := redactedContext{URL: c.URL}
	if c.Token != "" {
		r.Token = "***"
	}
	return r
}

// token is redacted whenever a Context is rendered; Save uses fileConfig to keep it
func (c Context) MarshalJSON() ([]byte, error) { return json.Marshal(c.redacted()) }

func (c Context) MarshalYAML() (any, error) { return c.redacted(), nil }

type Config struct {
	CurrentContext string             `yaml:"current-context"`
	Contexts       map[string]Context `yaml:"contexts"`
}

// fileConfig mirrors Config without Context's redacting marshalers.
type fileConfig struct {
	CurrentContext string                 `yaml:"current-context"`
	Contexts       map[string]fileContext `yaml:"contexts"`
}

type fileContext struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

// Save writes c to path with mode 0600; temp file + rename so a crash never leaves it half-written.
// Comments in an existing file are not preserved.
func (c *Config) Save(path string) error {
	f := fileConfig{CurrentContext: c.CurrentContext, Contexts: map[string]fileContext{}}
	for name, ctx := range c.Contexts {
		f.Contexts[name] = fileContext(ctx)
	}
	b, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".config-*.yaml") // created 0600
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck // no-op after a successful rename
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// DefaultPath is $XDG_CONFIG_HOME/sinkctl/config.yaml, else ~/.config/sinkctl/config.yaml.
func DefaultPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "sinkctl", "config.yaml"), nil
}

// Load reads the config file. A missing file yields an empty config.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path) //nolint:gosec // path is the user's own config file
	if errors.Is(err, fs.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &c, nil
}

// Resolve picks context name (or current-context when empty), then applies
// BUGSINK_URL / BUGSINK_TOKEN overrides from getenv.
func (c *Config) Resolve(name string, getenv func(string) string) (Context, error) {
	if name == "" {
		name = c.CurrentContext
	}
	var ctx Context
	if name != "" {
		var ok bool
		if ctx, ok = c.Contexts[name]; !ok {
			return Context{}, fmt.Errorf("context %q not found in config", name)
		}
	}
	if v := getenv(EnvURL); v != "" {
		ctx.URL = v
	}
	if v := getenv(EnvToken); v != "" {
		ctx.Token = v
	}
	if ctx.URL == "" || ctx.Token == "" {
		return Context{}, fmt.Errorf("no Bugsink URL/token: run `sinkctl config login`, or set %s and %s", EnvURL, EnvToken)
	}
	return ctx, nil
}
