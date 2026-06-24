package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

// errAfter is a sentinel returned by boundedWriter once its line budget is
// spent, so the framework's sink stops and gloo.Run returns — letting tests
// drive the otherwise-infinite default yes stream deterministically.
const errAfter Error = "bounded writer limit reached"

// Error is the package's single sentinel error type.
type Error string

func (e Error) Error() string { return string(e) }

// boundedWriter writes to buf but fails once writes exceed lines, bounding an
// infinite stream so the test cannot hang.
type boundedWriter struct {
	buf   *bytes.Buffer
	lines int
	seen  int
}

// Write records output and fails after the configured number of writes. Pointer
// receiver required: each call mutates the running write count.
func (w *boundedWriter) Write(p []byte) (int, error) {
	w.seen++
	if w.seen > w.lines {
		return 0, errAfter
	}
	return w.buf.Write(p)
}

func TestRun(t *testing.T) {
	cases := []struct {
		name       string
		version    string
		args       []string
		bound      int
		wantOut    string
		wantCode   int
		wantErrSub string
	}{
		{
			name:     "default y with count",
			args:     []string{"yes", "-n", "3"},
			wantOut:  "y\ny\ny\n",
			wantCode: 0,
		},
		{
			name:     "custom string from args with count",
			args:     []string{"yes", "-n", "2", "hello", "world"},
			wantOut:  "hello world\nhello world\n",
			wantCode: 0,
		},
		{
			name:     "long count alias",
			args:     []string{"yes", "--count", "1", "ok"},
			wantOut:  "ok\n",
			wantCode: 0,
		},
		{
			name:       "infinite default bounded by writer",
			args:       []string{"yes"},
			bound:      4,
			wantCode:   1,
			wantErrSub: "yes:",
		},
		{
			name:    "version flag reports injected version",
			version: "1.2.3",
			args:    []string{"yes", "--version"},
			wantOut: "yes version 1.2.3\n",
		},
		{
			name:       "unknown flag errors",
			args:       []string{"yes", "--nope"},
			wantCode:   1,
			wantErrSub: "yes:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			stdout := stdoutFor(tc.bound, &out)
			code := run(tc.version, tc.args, strings.NewReader(""), stdout, &errOut, afero.NewMemMapFs())

			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d (stderr=%q)", code, tc.wantCode, errOut.String())
			}
			if tc.wantErrSub == "" && out.String() != tc.wantOut {
				t.Fatalf("stdout = %q, want %q", out.String(), tc.wantOut)
			}
			if tc.wantErrSub != "" && !strings.Contains(errOut.String(), tc.wantErrSub) {
				t.Fatalf("stderr = %q, want substring %q", errOut.String(), tc.wantErrSub)
			}
		})
	}
}

// stdoutFor returns the plain buffer when unbounded, or a boundedWriter that
// fails after bound writes so an infinite stream terminates.
func stdoutFor(bound int, buf *bytes.Buffer) io.Writer {
	if bound == 0 {
		return buf
	}
	return &boundedWriter{buf: buf, lines: bound}
}

func Test_main(t *testing.T) {
	origExit, origRun := osExit, runCLI
	t.Cleanup(func() { osExit, runCLI = origExit, origRun })

	gotCode := -1
	osExit = func(code int) { gotCode = code }
	runCLI = func(string, []string, io.Reader, io.Writer, io.Writer, afero.Fs) int { return 7 }

	main()

	if gotCode != 7 {
		t.Fatalf("main propagated exit code %d, want 7", gotCode)
	}
}
