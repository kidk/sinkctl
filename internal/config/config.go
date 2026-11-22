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

// token is redacted whenever a Context is rendered; writing config.yaml needs a separate type
func (c Context) MarshalJSON() ([]byte, error) { return json.Marshal(c.redacted()) }

func (c Context) MarshalYAML() (any, error) { return c.redacted(), nil }

type Config struct {
	CurrentContext string             `yaml:"current-context"`
	Contexts       map[string]Context `yaml:"contexts"`
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
		return Context{}, fmt.Errorf("no Bugsink URL/token: set %s and %s or configure a context", EnvURL, EnvToken)
	}
	return ctx, nil
}
