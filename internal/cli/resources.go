package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kidk/sinkctl/internal/ops"
)

// groupCmd is a parent command; unknown subcommands get a usage error instead of help.
func groupCmd(use, short string, subs ...*cobra.Command) *cobra.Command {
	c := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return usageErr(fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath()))
			}
			return cmd.Help()
		},
	}
	c.AddCommand(subs...)
	return c
}

// opCmd builds ops from --context, runs fn and prints its result.
func opCmd[T any](g *globals, use, short string, nargs int, fn func(ctx context.Context, o *ops.Ops, args []string) (T, error)) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  usageArgs(cobra.ExactArgs(nargs)),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := g.ops()
			if err != nil {
				return err
			}
			v, err := fn(cmd.Context(), o, args)
			if err != nil {
				return err
			}
			return g.print(cmd, v)
		},
	}
}

// listCmd is opCmd for a paginated list, adding --cursor and --all.
func listCmd[T any](g *globals, use, short string, list func(ctx context.Context, o *ops.Ops, cursor string) (ops.Page[T], error)) *cobra.Command {
	var (
		all    bool
		cursor string
	)
	cmd := opCmd(g, use, short, 0, func(ctx context.Context, o *ops.Ops, _ []string) (ops.Page[T], error) {
		fetch := func(ctx context.Context, c string) (ops.Page[T], error) { return list(ctx, o, c) }
		if all {
			return ops.All(ctx, cursor, fetch)
		}
		return fetch(ctx, cursor)
	})
	cmd.Flags().StringVar(&cursor, "cursor", "", "start at next_cursor from a previous page")
	cmd.Flags().BoolVar(&all, "all", false, "follow cursors and return every page")
	return cmd
}

func intArg(name, s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, usageErr(fmt.Errorf("%s: want an integer, got %q", name, s))
	}
	return n, nil
}

func rangeFlags(cmd *cobra.Command, start, end *string) {
	cmd.Flags().StringVar(start, "start", "", "inclusive UTC hour, RFC 3339 (default 24h before end)")
	cmd.Flags().StringVar(end, "end", "", "exclusive UTC hour, RFC 3339 (default end of current hour)")
}
