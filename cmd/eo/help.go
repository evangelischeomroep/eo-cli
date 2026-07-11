package main

import (
	"fmt"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
)

func hasHelpFlag(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "--help" || a == "help" {
			return true
		}
	}
	return false
}

func printMainHelp() {
	var h helpDoc
	h.line(bold("eo") + dim(" — Evangelische Omroep developer CLI"))
	h.blank()
	h.line("  A small CLI for common EO developer tasks. Currently focused on")
	h.line("  Azure PIM: activating your own role and approving PIM requests")
	h.line("  from teammates.")
	h.section("USAGE")
	h.line("  eo <command> [flags] [arguments]")
	h.section("COMMANDS")
	h.cmd("pim", "Activate the Contributor role for 8h")
	h.cmd("pim approve", "List and approve pending PIM requests")
	h.cmd("version", "Print the current version")
	h.cmd("completion", "Output shell completion script")
	h.cmd("help", "Show help for a command")
	h.section("FLAGS")
	h.flag("-h, --help", "Show help")
	h.section("EXAMPLES")
	h.example("Activate Contributor role with default justification", "eo pim")
	h.example("Activate with a custom reason (shown in the Azure audit log)", `eo pim "deploying release 2.4"`)
	h.example("Approve pending PIM requests interactively", "eo pim approve")
	h.example("Approve everything pending in one go", "eo pim approve --all")
	h.section("REQUIREMENTS")
	h.line("  • Azure CLI (az) installed and logged in")
	h.line(fmt.Sprintf("  • Access to the %q subscription", azure.SubscriptionName))
	h.line("  • For approvals: you must be an approver on the relevant PIM policy")
	h.section("ENVIRONMENT")
	h.flag("NO_COLOR", "Disable colored output when set")
	h.blank()
	h.line(dim("Run ") + cyan("eo <command> --help") + dim(" for details on a specific command."))
	h.print()
}

func printPimHelp() {
	var h helpDoc
	h.line(bold("eo pim") + dim(" — Activate the Contributor role on Azure"))
	h.blank()
	h.line("  Activates the Contributor role on the " + cyan(azure.SubscriptionName) + " subscription for")
	h.line("  8 hours. The optional reason is stored in the Azure PIM audit log.")
	h.section("USAGE")
	h.line("  eo pim [reason]")
	h.section("ARGUMENTS")
	h.flag("reason", "Justification for the activation (optional)")
	h.section("EXAMPLES")
	h.example("", "eo pim")
	h.example("", `eo pim "deploying release 2.4"`)
	h.section("NOTES")
	h.line("  • Activation lasts 8 hours from the moment of activation")
	h.line("  • If the role is already active you get a warning, no error")
	h.print()
}

func printPimApproveHelp() {
	var h helpDoc
	h.line(bold("eo pim approve") + dim(" — Approve pending PIM requests"))
	h.blank()
	h.line("  Lists PIM role activation requests where you are an approver and")
	h.line("  lets you approve them interactively or all at once with --all.")
	h.section("USAGE")
	h.line("  eo pim approve [--all] [justification]")
	h.section("FLAGS")
	h.flag("--all", "Approve all pending requests without prompting")
	h.section("ARGUMENTS")
	h.flag("justification", `Reason attached to each approval (default "Approved via eo-cli")`)
	h.section("EXAMPLES")
	h.example("Interactive selection — pick which requests to approve", "eo pim approve")
	h.example("Approve everything at once", "eo pim approve --all")
	h.example("Approve all with a custom justification", `eo pim approve --all "sprint review batch"`)
	h.section("NOTES")
	h.line("  • Only shows requests where you are listed as an approver")
	h.line(`  • Interactive input accepts space-separated numbers or "all"`)
	h.print()
}
