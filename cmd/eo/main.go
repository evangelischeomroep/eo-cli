package main

import (
	"fmt"
	"os"
)

// version is set at build time via -ldflags "-X main.version=x.y.z"
var version = "dev"

func main() {
	updateCh := startUpdateCheck()
	err := run(os.Args[1:])
	printUpdateNotice(updateCh, version)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("✗ ")+err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printMainHelp()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printMainHelp()
		return nil
	case "version", "-v", "--version":
		fmt.Printf("eo %s\n", version)
		return nil
	case "deploy":
		if hasHelpFlag(args[1:]) {
			printDeployHelp()
			return nil
		}
		return cmdDeploy(args[1:])
	case "pim":
		if len(args) >= 2 && args[1] == "approve" {
			if hasHelpFlag(args[2:]) {
				printPimApproveHelp()
				return nil
			}
			return cmdPimApprove(args[2:])
		}
		if len(args) >= 2 && args[1] == "status" {
			if hasHelpFlag(args[2:]) {
				printPimStatusHelp()
				return nil
			}
			return cmdPimStatus()
		}
		if hasHelpFlag(args[1:]) {
			printPimHelp()
			return nil
		}
		return cmdPimRequest(args[1:])
	case "completion":
		shell := ""
		if len(args) >= 2 {
			shell = args[1]
		}
		return cmdCompletion(shell)
	default:
		printMainHelp()
		return fmt.Errorf("unknown command: %s", args[0])
	}
}
