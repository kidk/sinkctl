package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/kidk/sinkctl/internal/config"
	"github.com/kidk/sinkctl/internal/ops"
)

type contextInfo struct {
	Context string `json:"context"`
	URL     string `json:"url"`
}

func (c contextInfo) Table() ([]string, [][]string) {
	return []string{"CONTEXT", "URL"}, [][]string{{c.Context, c.URL}}
}

func newConfigCmd(g *globals) *cobra.Command {
	var (
		url        string
		tokenStdin bool
		noVerify   bool
	)
	login := &cobra.Command{
		Use:   "login [name]",
		Short: "Save a Bugsink URL and token as a context and switch to it",
		Long: "Save a Bugsink URL and token as a context and switch to it.\n\n" +
			"On a terminal, prompts for whatever the arguments leave out. " +
			"Without one (or with --token-stdin), name and --url are required.",
		Args: usageArgs(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) == 1 {
				name = args[0]
			}
			p := newPrompter(cmd)
			if !p.tty || tokenStdin {
				switch {
				case url == "":
					return usageErr(errors.New("--url is required when not prompting (no terminal, or --token-stdin)"))
				case name == "":
					return usageErr(errors.New("context name is required when not prompting (no terminal, or --token-stdin)"))
				case !tokenStdin:
					return usageErr(errors.New("stdin is not a terminal; pipe the token with --token-stdin"))
				}
			}
			c, err := loginFlow(cmd.Context(), p, name, url, tokenStdin, !noVerify)
			if err != nil {
				return err
			}
			return saveContext(cmd, g, c.name, &c.Context)
		},
	}
	login.Flags().StringVar(&url, "url", "", "Bugsink base URL, like https://bugsink.example.com (prompted if omitted)")
	login.Flags().BoolVar(&tokenStdin, "token-stdin", false, "read the token from stdin instead of prompting")
	login.Flags().BoolVar(&noVerify, "no-verify", false, "save without checking the token against Bugsink")

	use := &cobra.Command{
		Use:   "use <name>",
		Short: "Switch the current context",
		Args:  usageArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return saveContext(cmd, g, args[0], nil)
		},
	}

	return groupCmd("config", "Manage saved Bugsink contexts", login, use)
}

// saveContext stores c under name (nil: name must exist) and makes it current.
func saveContext(cmd *cobra.Command, g *globals, name string, c *config.Context) error {
	path, cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if cfg.Contexts == nil {
		cfg.Contexts = map[string]config.Context{}
	}
	if c != nil {
		cfg.Contexts[name] = *c
	}
	saved, ok := cfg.Contexts[name]
	if !ok {
		return &ops.Error{Kind: ops.KindAuth, Message: fmt.Sprintf("context %q not found in config", name)}
	}
	cfg.CurrentContext = name
	if err := cfg.Save(path); err != nil {
		return err
	}
	return g.print(cmd, contextInfo{Context: name, URL: saved.URL})
}

const loginTries = 3

type namedContext struct {
	name string
	config.Context
}

// loginFlow fills in name, URL and token, prompting for missing ones; prompted values get loginTries attempts.
// Callers ensure p.tty when anything must be prompted.
func loginFlow(ctx context.Context, p *prompter, name, url string, tokenStdin, check bool) (namedContext, error) {
	c := namedContext{name: name, Context: config.Context{URL: strings.TrimRight(url, "/")}}
	if c.URL == "" {
		err := p.retry(func() (err error) {
			if c.URL, err = p.line("Bugsink URL: "); err != nil {
				return err
			}
			c.URL = strings.TrimRight(c.URL, "/")
			if !check {
				return nonEmpty("URL", c.URL)
			}
			return probeURL(ctx, c.URL)
		})
		if err != nil {
			return c, err
		}
	}
	if c.name == "" {
		host := ""
		if u, err := neturl.Parse(c.URL); err == nil {
			host = u.Hostname()
		}
		err := p.retry(func() (err error) {
			if c.name, err = p.line(fmt.Sprintf("Context name [%s]: ", host)); err != nil {
				return err
			}
			if c.name == "" {
				c.name = host
			}
			return nonEmpty("context name", c.name)
		})
		if err != nil {
			return c, err
		}
	}
	if tokenStdin {
		line, err := p.r.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return c, err
		}
		if c.Token = strings.TrimSpace(line); c.Token == "" {
			return c, usageErr(errors.New("token is empty"))
		}
		if check {
			return c, verify(ctx, c.Context)
		}
		return c, nil
	}
	err := p.retry(func() (err error) {
		if c.Token, err = p.secret("Token: "); err != nil {
			return err
		}
		if err := nonEmpty("token", c.Token); err != nil || !check {
			return err
		}
		return verify(ctx, c.Context)
	})
	return c, err
}

func nonEmpty(what, v string) error {
	if v == "" {
		return usageErr(fmt.Errorf("%s is empty", what))
	}
	return nil
}

// prompter reads answers from stdin and writes prompts to stderr so stdout stays machine-readable.
type prompter struct {
	r        *bufio.Reader
	w        io.Writer
	tty      bool
	password func() (string, error) // hidden read; tests swap it via newPrompter
}

// newPrompter is a var so tests can fake a terminal.
var newPrompter = func(cmd *cobra.Command) *prompter {
	in := cmd.InOrStdin()
	p := &prompter{r: bufio.NewReader(in), w: cmd.ErrOrStderr()}
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		p.tty = true
		p.password = func() (string, error) {
			b, err := term.ReadPassword(int(f.Fd()))
			return string(b), err
		}
	}
	return p
}

func (p *prompter) line(prompt string) (string, error) {
	fmt.Fprint(p.w, prompt)
	s, err := p.r.ReadString('\n')
	if err != nil && (s == "" || !errors.Is(err, io.EOF)) {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

func (p *prompter) secret(prompt string) (string, error) {
	fmt.Fprint(p.w, prompt)
	s, err := p.password()
	fmt.Fprintln(p.w)
	return strings.TrimSpace(s), err
}

// retry runs fn up to loginTries times on *ops.Error; any other error (EOF, cancel) aborts at once.
// The last failure is returned, not printed, so Run reports it once.
func (p *prompter) retry(fn func() error) error {
	for i := 1; ; i++ {
		err := fn()
		var e *ops.Error
		if !errors.As(err, &e) || i == loginTries {
			return err
		}
		fmt.Fprintln(p.w, "Error:", e.Message)
	}
}

// probeURL checks, without a token, that url serves the Bugsink API: a JSON 200, 401 or 403 on the projects endpoint.
func probeURL(ctx context.Context, url string) error {
	if err := nonEmpty("URL", url); err != nil {
		return err
	}
	endpoint := url + "/api/canonical/0/projects/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return usageErr(fmt.Errorf("invalid URL %q: %w", url, err))
	}
	res, err := httpClient().Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		return &ops.Error{Kind: ops.KindNetwork, Message: fmt.Sprintf("cannot reach %s: %v", url, err)}
	}
	defer res.Body.Close() //nolint:errcheck // read-only probe
	body, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	switch res.StatusCode {
	case http.StatusOK, http.StatusUnauthorized, http.StatusForbidden:
		if json.Valid(body) {
			return nil
		}
	}
	return &ops.Error{Kind: ops.KindAPI, Message: fmt.Sprintf("%s does not look like Bugsink: GET %s returned %d", url, endpoint, res.StatusCode), Status: res.StatusCode}
}

func verify(ctx context.Context, c config.Context) error {
	o, err := ops.New(c, httpClient())
	if err != nil {
		return err
	}
	_, err = o.ListProjects(ctx, ops.ListProjectsIn{})
	return err
}
