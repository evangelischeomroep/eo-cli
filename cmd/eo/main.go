package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X main.version=x.y.z"
var version = "dev"

func main() {
	args := os.Args[1:]
	var updateCh <-chan string
	if len(args) == 0 || !slices.Contains([]string{
		"completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd,
	}, args[0]) {
		updateCh = startUpdateCheck()
	}
	err := newRootCommand().Execute()
	if updateCh != nil {
		printUpdateNotice(updateCh, version)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, red("✗ ")+err.Error())
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "eo",
		Short:         "Evangelische Omroep developer CLI",
		Long:          "Evangelische Omroep developer CLI for Azure PIM and EOchat.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	releaseLink := ""
	if version != "dev" {
		releaseLink = dim("https://github.com/evangelischeomroep/eo-cli/releases/tag/v"+strings.TrimPrefix(version, "v")) + "\n"
	}
	root.SetVersionTemplate("{{.Name}} {{.Version}}\n" + releaseLink)
	root.AddCommand(newAskCommand(cmdAsk), newPimCommand(), newMCPCommand(),
		&cobra.Command{
			Use:   "whoami",
			Short: "Show the current Azure user and subscription",
			Args:  cobra.NoArgs,
			RunE:  func(*cobra.Command, []string) error { return cmdWhoami() },
		},
		&cobra.Command{
			Use:   "version",
			Short: "Print the current version",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "eo %s\n%s", version, releaseLink)
				return err
			},
		},
	)
	return root
}
