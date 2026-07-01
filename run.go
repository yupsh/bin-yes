package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	command "github.com/gloo-foo/cmd-yes"
	gloo "github.com/gloo-foo/framework"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

const name = "yes"

const flagCount = "count"

// usageText is the command's multi-line usage synopsis, shown in --help.
// cli/v3 indents the whole block by 3 spaces, so these lines are flush-left to
// stay aligned in the rendered output.
const usageText = `yes [OPTIONS] [STRING]...

repeatedly output a line with all specified STRING(s), or 'y'.`

// init replaces urfave/cli's default --version/-v flag with a --version-only
// flag, freeing the single-letter -v for command flags (e.g. grep -v) while
// still exposing the injected build version.
func init() {
	cli.VersionFlag = &cli.BoolFlag{Name: "version", Usage: "print version information and exit"}
}

// run builds and executes the yes CLI against the injected version, I/O, and
// filesystem, returning the process exit code. yes does not read stdin or the
// filesystem; both are injected for a uniform, testable wiring shape.
func run(version string, args []string, _ io.Reader, stdout, stderr io.Writer, _ afero.Fs) int {
	cmd := newCommand(version, stdout)
	cmd.Writer = stdout
	cmd.ErrWriter = stderr
	if err := cmd.Run(context.Background(), args); err != nil {
		_, _ = fmt.Fprintf(stderr, name+": %v\n", err)
		return 1
	}
	return 0
}

func newCommand(version string, stdout io.Writer) *cli.Command {
	return &cli.Command{
		Name:            name,
		Version:         version,
		Usage:           "output a string repeatedly until killed",
		UsageText:       usageText,
		HideHelpCommand: true,
		// Keep exit handling in run() rather than letting urfave/cli call
		// os.Exit, so the exit code stays testable.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    flagCount,
				Aliases: []string{"n"},
				Usage:   "output COUNT lines instead of repeating forever",
			},
		},
		Action: action(stdout),
	}
}

func action(stdout io.Writer) cli.ActionFunc {
	return func(_ context.Context, cmd *cli.Command) error {
		_, err := gloo.Run(command.Yes(options(cmd)...), gloo.ByteWriteTo(stdout))
		return err
	}
}

func options(cmd *cli.Command) []any {
	opts := []any{command.YesText(text(cmd))}
	if cmd.IsSet(flagCount) {
		opts = append(opts, command.YesCount(cmd.Int(flagCount)))
	}
	return opts
}

func text(cmd *cli.Command) string {
	if cmd.NArg() == 0 {
		return "y"
	}
	return strings.Join(cmd.Args().Slice(), " ")
}
