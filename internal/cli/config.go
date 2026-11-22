package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
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
		Use:   "login <name>",
		Short: "Save a Bugsink URL and token as a context and switch to it",
		Args:  usageArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			if url == "" {
				return usageErr(errors.New("--url is required"))
			}
			token, err := readToken(cmd, tokenStdin)
			if err != nil {
				return err
			}
			c := config.Context{URL: strings.TrimRight(url, "/"), Token: token}
			if !noVerify {
				if err := verify(cmd.Context(), c); err != nil {
					return err
				}
			}
			return saveContext(cmd, g, args[0], &c)
		},
	}
	login.Flags().StringVar(&url, "url", "", "Bugsink base URL, like https://bugsink.example.com (required)")
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

// readToken prompts on a terminal; otherwise --token-stdin is required so a pipe is never read by surprise.
func readToken(cmd *cobra.Command, fromStdin bool) (string, error) {
	in := cmd.InOrStdin()
	f, isFile := in.(*os.File)
	var token string
	switch {
	case fromStdin:
		line, err := bufio.NewReader(in).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		token = line
	case isFile && term.IsTerminal(int(f.Fd())):
		fmt.Fprint(cmd.ErrOrStderr(), "Token: ")
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(cmd.ErrOrStderr())
		if err != nil {
			return "", err
		}
		token = string(b)
	default:
		return "", usageErr(errors.New("stdin is not a terminal; pipe the token with --token-stdin"))
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", usageErr(errors.New("token is empty"))
	}
	return token, nil
}

func verify(ctx context.Context, c config.Context) error {
	o, err := ops.New(c, httpClient())
	if err != nil {
		return err
	}
	_, err = o.ListProjects(ctx, ops.ListProjectsIn{})
	return err
}
