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
	path, err := config.DefaultPath()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	c, err := cfg.Resolve(g.context, os.Getenv)
	if err != nil {
		return nil, &ops.Error{Kind: ops.KindAuth, Message: err.Error()}
	}
	return ops.New(c, &http.Client{Timeout: 30 * time.Second})
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
	root := &cobra.Command{
		Use:           "sinkctl",
		Short:         "CLI and MCP server for Bugsink",
		SilenceUsage:  true,
		SilenceErrors: true,
		// RunE + ArbitraryArgs so unknown subcommands land here and get a usage exit code
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return usageErr(fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath()))
			}
			return cmd.Help()
		},
	}
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageErr(err) })
	root.PersistentFlags().StringVar(&g.context, "context", "", "config context to use (default: current-context)")
	root.PersistentFlags().StringVarP(&g.output, "output", "o", string(output.Auto), "output format: auto, json, yaml, table")

	root.AddCommand(
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

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	g := &globals{}
	root := newRootCmd(g)
	root.SetArgs(args)
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
	return run(ctx, os.Args[1:], os.Stdout, os.Stderr)
}
