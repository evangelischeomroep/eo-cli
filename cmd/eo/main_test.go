package main

import (
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestPimRequiresExplicitActivation(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{name: "group help", args: []string{}},
		{name: "misspelled command", args: []string{"statuz"}, wantErr: true},
		{name: "old justification form", args: []string{"deploying release 2.4"}, wantErr: true},
		{name: "explicit activation", args: []string{"activate", "deploying release 2.4"}, want: "activate"},
		{name: "explicit approval", args: []string{"approve", "--all", "sprint review"}, want: "approve"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cmd := newPimCommand()
			var called string
			for _, child := range cmd.Commands() {
				child.RunE = func(selected *cobra.Command, _ []string) error {
					called = selected.Name()
					return nil
				}
			}
			cmd.SetArgs(test.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			if err := cmd.Execute(); (err != nil) != test.wantErr || called != test.want {
				t.Fatalf("command = %q, error = %v, want command %q and error = %t", called, err, test.want, test.wantErr)
			}
		})
	}
}

func TestGeneratedHelpAndCompletions(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"root help", []string{}, "Evangelische Omroep developer CLI"},
		{"nested help", []string{"help", "pim", "approve"}, "--all"},
		{"EOchat model help", []string{"ask", "models", "--help"}, "eo ask models"},
		{"MCP install help", []string{"mcp", "install", "--help"}, "--project"},
		{"Bash script", []string{"completion", "bash"}, "# bash completion V2 for eo"},
		{"Zsh script", []string{"completion", "zsh"}, "#compdef eo"},
		{"Fish script", []string{"completion", "fish"}, "# fish completion for eo"},
		{"approval flag", []string{"__completeNoDesc", "pim", "approve", "--a"}, "--all\n"},
		{"project flag", []string{"__completeNoDesc", "mcp", "install", "--p"}, "--project\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cmd := newRootCommand()
			var out strings.Builder
			cmd.SetArgs(test.args)
			cmd.SetOut(&out)
			cmd.SetErr(io.Discard)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), test.want) {
				t.Errorf("output does not contain %q", test.want)
			}
		})
	}
}
