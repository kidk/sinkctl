// Package cli wires sinkctl ops into cobra commands.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/kidk/sinkctl/internal/config"
	sinkmcp "github.com/kidk/sinkctl/internal/mcp"
	"github.com/kidk/sinkctl/internal/ops"
	"github.com/kidk/sinkctl/internal/output"
)

type globals struct {
	context string
	output  string
}

// print renders v to cmd's stdout using the -o flag, auto-detecting TTY.
func (g *globals) print(cmd *cobra.Command, v any) error {
	f, err := output.ParseFormat(g.output)
	if err != nil {
		return usageErr(err)
	}
	w := cmd.OutOrStdout()
	return output.Write(w, output.Resolve(f, output.IsTerminal(w)), v)
}

// ops resolves --context plus env into a client; built per command so `version` needs no creds.
func (g *globals) ops() (*ops.Ops, error) {
	_, cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	c, err := cfg.Resolve(g.context, os.Getenv)
	if err != nil {
		return nil, &ops.Error{Kind: ops.KindAuth, Message: err.Error()}
	}
	return ops.New(c, httpClient())
}

func loadConfig() (path string, cfg *config.Config, err error) {
	if path, err = config.DefaultPath(); err != nil {
		return "", nil, err
	}
	cfg, err = config.Load(path)
	return path, cfg, err
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func usageErr(err error) error {
	return &ops.Error{Kind: ops.KindUsage, Message: err.Error()}
}

func usageArgs(fn cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := fn(cmd, args); err != nil {
			return usageErr(err)
		}
		return nil
	}
}

func newRootCmd(g *globals) *cobra.Command {
	root := groupCmd("sinkctl", "CLI and MCP server for Bugsink")
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageErr(err) })
	root.PersistentFlags().StringVar(&g.context, "context", "", "config context to use (default: current-context)")
	root.PersistentFlags().StringVarP(&g.output, "output", "o", string(output.Auto), "output format: auto, json, yaml, table")

	root.AddCommand(
		newProjectsCmd(g),
		newIssuesCmd(g),
		newEventsCmd(g),
		newConfigCmd(g),
		&cobra.Command{
			Use:   "version",
			Short: "Show build information",
			Args:  usageArgs(cobra.NoArgs),
			RunE: func(cmd *cobra.Command, _ []string) error {
				return g.print(cmd, ops.Version())
			},
		},
		&cobra.Command{
			Use:   "mcp",
			Short: "Run an MCP server on stdio",
			Args:  usageArgs(cobra.NoArgs),
			RunE: func(cmd *cobra.Command, _ []string) error {
				o, err := g.ops()
				if err != nil {
					return err
				}
				return sinkmcp.NewServer(o).Run(cmd.Context(), &mcp.StdioTransport{})
			},
		},
	)
	return root
}

var exitCodes = map[ops.Kind]int{
	ops.KindUsage:    2,
	ops.KindAuth:     3,
	ops.KindNotFound: 4,
	ops.KindAPI:      5,
	ops.KindNetwork:  5,
}

// Run executes args and returns the exit code; Execute wires it to the process.
func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	g := &globals{}
	root := newRootCmd(g)
	root.SetArgs(args)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)

	err := root.ExecuteContext(ctx)
	if err == nil {
		return 0
	}
	if errors.Is(err, context.Canceled) && ctx.Err() != nil {
		return 130 // interrupted; shell convention for SIGINT
	}

	var e *ops.Error
	if !errors.As(err, &e) {
		e = &ops.Error{Kind: ops.KindError, Message: err.Error()}
	}
	if f, ferr := output.ParseFormat(g.output); ferr == nil && output.Resolve(f, output.IsTerminal(stdout)) == output.JSON {
		_ = json.NewEncoder(stderr).Encode(map[string]*ops.Error{"error": e})
	} else {
		fmt.Fprintln(stderr, "Error:", e.Message)
	}
	if code, ok := exitCodes[e.Kind]; ok {
		return code
	}
	return 1
}

// Execute runs the CLI and returns the process exit code.
func Execute(ctx context.Context) int {
	return Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
}
