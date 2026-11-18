package main

import (
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAskArgumentParsing(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		want    askOptions
		wantErr string
	}{
		{
			name: "flags and prompt",
			args: []string{"-m", "models", "-c", "--system=be brief", "--web", "what", "is", "this?", "--raw"},
			want: askOptions{model: "models", cont: true, system: "be brief", web: true, raw: true, prompt: "what is this?"},
		},
		{
			name: "repeatable values preserve commas",
			args: []string{"-f", "a,b.pdf", "--file=c.pdf", "--knowledge=News,Sport", "-k", "Handboek", "review"},
			want: askOptions{files: []string{"a,b.pdf", "c.pdf"}, knowledge: []string{"News,Sport", "Handboek"}, prompt: "review"},
		},
		{name: "chat or piped input", args: []string{}},
		{name: "help as text", args: []string{"help", "me"}, want: askOptions{prompt: "help me"}},
		{name: "literal subcommand", args: []string{"--", "models"}, want: askOptions{prompt: "models"}},
		{name: "literal flag", args: []string{"--", "--help"}, want: askOptions{prompt: "--help"}},
		{name: "unknown flag", args: []string{"--bogus"}, wantErr: "unknown flag"},
		{name: "missing value", args: []string{"-m"}, wantErr: "needs an argument"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var got askOptions
			called := false
			cmd := &cobra.Command{Use: "eo"}
			cmd.AddCommand(newAskCommand(func(opts askOptions) error {
				got, called = opts, true
				return nil
			}))
			cmd.SetArgs(append([]string{"ask"}, test.args...))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			err := cmd.Execute()
			if test.wantErr != "" {
				if called || err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("called = %t, error = %v, want %q", called, err, test.wantErr)
				}
				return
			}
			if err != nil || !called || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("called = %t, options = %+v, error = %v, want %+v", called, got, err, test.want)
			}
		})
	}
}

func TestMDWriterStyled(t *testing.T) {
	old := useColor
	useColor = true
	defer func() { useColor = old }()

	var out strings.Builder
	w := newMDWriter(&out, true)
	w.write("# Tit")
	w.write("le\n- item with **bold** and `code`\n```go\nfmt.Println()\n```\ntrailing")
	w.flush()
	got := out.String()

	if !strings.Contains(got, ansiBold+"Title"+ansiReset) {
		t.Errorf("heading not bold: %q", got)
	}
	if !strings.Contains(got, "• item") || !strings.Contains(got, ansiBold+"bold"+ansiReset) || !strings.Contains(got, ansiCyan+"code"+ansiReset) {
		t.Errorf("inline styling missing: %q", got)
	}
	if !strings.Contains(got, ansiCyan+"fmt.Println()"+ansiReset) {
		t.Errorf("code block not styled: %q", got)
	}
	if !strings.HasSuffix(got, "trailing") {
		t.Errorf("trailing partial line lost: %q", got)
	}
}

func TestMDWriterRawPassthrough(t *testing.T) {
	var out strings.Builder
	w := newMDWriter(&out, false)
	in := "# Title\n- **x** `y`\n```\ncode\n```\nend"
	w.write(in)
	w.flush()
	if out.String() != in {
		t.Errorf("raw mode altered output: %q", out.String())
	}
}
