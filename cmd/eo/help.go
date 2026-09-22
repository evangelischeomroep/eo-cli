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
	h.line("  A small CLI for common EO developer tasks.")
	h.section("USAGE")
	h.line("  eo <command> [flags] [arguments]")
	h.section("COMMANDS")
	h.cmd("deploy", "Deploy Function Apps to test or prod")
	h.cmd("pim", "Activate the Contributor role for 8h")
	h.cmd("pim approve", "List and approve pending PIM requests")
	h.cmd("pim status", "Show if your Contributor role is active")
	h.cmd("whoami", "Show the current Azure user and subscription")
	h.cmd("ask", "Ask EOchat (chat.eo.nl) from the terminal")
	h.cmd("mcp", "Expose EOchat to Claude Code as an MCP server")
	h.cmd("version", "Print the current version")
	h.cmd("completion", "Output shell completion script")
	h.cmd("help", "Show help for a command")
	h.section("FLAGS")
	h.flag("-h, --help", "Show help")
	h.section("EXAMPLES")
	h.example("Deploy selected apps to test", "eo deploy test")
	h.example("Deploy all apps to test", "eo deploy test --all")
	h.example("Activate Contributor role with default justification", "eo pim")
	h.example("Approve pending PIM requests interactively", "eo pim approve")
	h.example("Ask EOchat a question", `eo ask "hoe werkt onze deploy pipeline?"`)
	h.example("Review a diff with EOchat", `git diff | eo ask "review this change"`)
	h.section("REQUIREMENTS")
	h.line("  • Azure CLI (az) installed and logged in")
	h.line(fmt.Sprintf("  • Access to the %q subscription", azure.SubscriptionName))
	h.line("  • For approvals: you must be an approver on the relevant PIM policy")
	h.line("  • For ask/mcp: an EOchat API key (eo ask login)")
	h.section("ENVIRONMENT")
	h.flag("NO_COLOR", "Disable colored output when set")
	h.flag("EOCHAT_API_KEY", "EOchat API key (overrides `eo ask login`)")
	h.blank()
	h.line(dim("Run ") + cyan("eo <command> --help") + dim(" for details on a specific command."))
	h.print()
}

func printDeployHelp() {
	var h helpDoc
	h.line(bold("eo deploy") + dim(" — Deploy Function Apps to test or prod"))
	h.blank()
	h.line("  Lists all Function Apps in the target environment's resource group,")
	h.line("  lets you pick which ones to deploy, and triggers the pipeline stage.")
	h.section("USAGE")
	h.line("  eo deploy [test|prod] [--all] [--status]")
	h.section("ARGUMENTS")
	h.flag("test|prod", "Target environment (default: test)")
	h.section("FLAGS")
	h.flag("--all, -a", "Deploy all apps without interactive selection")
	h.flag("--status, -s", "Watch deployment status until all apps complete")
	h.section("EXAMPLES")
	h.example("Interactive selection for test", "eo deploy test")
	h.example("Deploy all apps to test", "eo deploy test --all")
	h.example("Deploy and watch status", "eo deploy test -s")
	h.example("Deploy selected apps to prod", "eo deploy prod")
	h.section("NOTES")
	h.line("  • Prod deployments require the test stage to be completed first")
	h.line("  • Prod will ask for confirmation before approving")
	h.print()
}

func printWhoamiHelp() {
	var h helpDoc
	h.line(bold("eo whoami") + dim(" — Show the current Azure user and subscription"))
	h.blank()
	h.line("  Displays the name, email, and active subscription of the currently")
	h.line("  logged-in Azure account.")
	h.section("USAGE")
	h.line("  eo whoami")
	h.section("EXAMPLES")
	h.example("", "eo whoami")
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

func printPimStatusHelp() {
	var h helpDoc
	h.line(bold("eo pim status") + dim(" — Show Contributor role status"))
	h.blank()
	h.line("  Shows whether the Contributor role is currently active on the")
	h.line("  " + cyan(azure.SubscriptionName) + " subscription and how much time remains.")
	h.section("USAGE")
	h.line("  eo pim status")
	h.section("EXAMPLES")
	h.example("", "eo pim status")
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

// askWantsHelp is stricter than hasHelpFlag: a question may legitimately
// contain the word "help", so only a leading -h/--help or a lone "help" counts.
func askWantsHelp(args []string) bool {
	if len(args) == 0 {
		return false
	}
	if args[0] == "-h" || args[0] == "--help" {
		return true
	}
	return len(args) == 1 && args[0] == "help"
}

func printAskHelp() {
	var h helpDoc
	h.line(bold("eo ask") + dim(" — Ask EOchat from the terminal"))
	h.blank()
	h.line("  Talks to EOchat (" + cyan("https://chat.eo.nl") + "), the EO AI assistant, using your")
	h.line("  personal API key. Answers stream to the terminal. Without a question")
	h.line("  you get an interactive chat; piped input is sent along as context.")
	h.section("USAGE")
	h.line("  eo ask [flags] [question]")
	h.line("  eo ask login | logout | status | models | model [name] | knowledge")
	h.section("FLAGS")
	h.flag("-m, --model", "Model to use (ID, name, or unique part of it)")
	h.flag("-c, --continue", "Continue the previous conversation")
	h.flag("-s, --system", "System prompt for this conversation")
	h.flag("-f, --file", "Upload a file and use it as context (repeatable)")
	h.flag("-k, --knowledge", "Use an EOchat knowledge base (repeatable)")
	h.flag("--web", "Let EOchat search the web (if enabled on the server)")
	h.flag("--raw", "No styling — plain text, handy for scripts")
	h.section("SUBCOMMANDS")
	h.cmd("login", "Store your EOchat API key and pick a default model")
	h.cmd("logout", "Remove the stored API key")
	h.cmd("status", "Show server, user, default model and last conversation")
	h.cmd("models", "List the models you can use")
	h.cmd("model [name]", "Set the default model (interactive without a name)")
	h.cmd("knowledge", "List knowledge bases you can use with -k")
	h.section("EXAMPLES")
	h.example("First time: store your API key", "eo ask login")
	h.example("One question", `eo ask "wat is het verschil tussen een Function App en een Container App?"`)
	h.example("Interactive chat", "eo ask")
	h.example("Pipe context in", `git diff | eo ask "review this change, focus on bugs"`)
	h.example("Explain an error", `make build 2>&1 | eo ask "what went wrong?"`)
	h.example("Keep going where you left off", `eo ask -c "and how do I test that?"`)
	h.example("Use a specific model and a knowledge base", `eo ask -m gpt -k "EO Handboek" "hoe vraag ik PIM aan?"`)
	h.example("Ask about a document", `eo ask -f ./architectuur.pdf "summarize the trade-offs"`)
	h.section("INTERACTIVE COMMANDS")
	h.line("  /model [name]  /models  /knowledge  /use <kb>  /file <path>  /system <text>  /web  /new  /quit")
	h.line("  End a line with " + cyan("\\") + " to continue on the next line. Ctrl+C stops an answer.")
	h.section("ENVIRONMENT")
	h.flag("EOCHAT_API_KEY", "API key (overrides the stored one)")
	h.flag("EOCHAT_MODEL", "Default model (overrides the stored one)")
	h.flag("EOCHAT_URL", "Server URL (default https://chat.eo.nl)")
	h.section("NOTES")
	h.line("  • Create a key in EOchat: Settings → Account → API keys")
	h.line("  • Config lives in ~/.config/eo/eochat.json (mode 0600)")
	h.line("  • Conversations are kept locally, not in the EOchat web history")
	h.print()
}

func printMcpHelp() {
	var h helpDoc
	h.line(bold("eo mcp") + dim(" — Expose EOchat to Claude Code as an MCP server"))
	h.blank()
	h.line("  Runs a Model Context Protocol server on stdin/stdout so AI coding")
	h.line("  tools (Claude Code, Cursor, ...) can consult EOchat models and")
	h.line("  knowledge bases. Uses the same login as " + cyan("eo ask") + ".")
	h.section("USAGE")
	h.line("  eo mcp                     Run the server (started by the MCP client, not by you)")
	h.line("  eo mcp install             Register with Claude Code for your user")
	h.line("  eo mcp install --project   Register in the current project's .mcp.json")
	h.section("TOOLS")
	tool := func(name, desc string) { h.line("  " + cyan(pad(name, 24)) + "  " + desc) }
	tool("eochat_ask", "Ask a question, optionally with model + knowledge bases")
	tool("eochat_list_models", "List available models")
	tool("eochat_list_knowledge", "List available knowledge bases")
	h.section("EXAMPLES")
	h.example("Register with Claude Code", "eo mcp install")
	h.example("Register by hand", "claude mcp add --scope user eochat -- eo mcp")
	h.example("Then, inside Claude Code", `"Use eochat_ask to find out how EO names Azure resources."`)
	h.section("NOTES")
	h.line("  • Run " + cyan("eo ask login") + " first; the server reads the same API key")
	h.line("  • Each eochat_ask call is a fresh conversation")
	h.print()
}
