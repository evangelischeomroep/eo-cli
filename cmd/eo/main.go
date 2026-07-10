package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/evangelischeomroep/eo-cli/internal/pim"
)

// version is set at build time via -ldflags "-X main.version=x.y.z"
var version = "dev"

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
	h.line(fmt.Sprintf("  • Access to the %q subscription", pim.SubscriptionName))
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
	h.line("  Activates the Contributor role on the " + cyan(pim.SubscriptionName) + " subscription for")
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

func cmdCompletion(shell string) error {
	switch shell {
	case "zsh":
		fmt.Print(zshCompletion)
	case "bash":
		fmt.Print(bashCompletion)
	default:
		fmt.Fprintln(os.Stderr, bold("eo completion")+" — Output shell completion script")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, bold("USAGE"))
		fmt.Fprintln(os.Stderr, "  eo completion <shell>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, bold("SHELLS"))
		fmt.Fprintln(os.Stderr, "  zsh   bash")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, bold("SETUP"))
		fmt.Fprintln(os.Stderr, dim("  # zsh — add to ~/.zshrc:"))
		fmt.Fprintln(os.Stderr, `  source <(eo completion zsh)`)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, dim("  # bash — add to ~/.bashrc:"))
		fmt.Fprintln(os.Stderr, `  source <(eo completion bash)`)
		if shell != "" {
			return fmt.Errorf("unknown shell %q — supported: zsh, bash", shell)
		}
	}
	return nil
}

const zshCompletion = `_eo() {
  local state

  _arguments \
    '1: :->command' \
    '*: :->args'

  case $state in
    command)
      local -a commands
      commands=(
        'pim:Activate the Contributor role for 8h'
        'version:Print the current version'
        'completion:Output shell completion script'
        'help:Show help for a command'
      )
      _describe 'command' commands
      ;;
    args)
      case $words[2] in
        pim)
          local -a pim_cmds
          pim_cmds=('approve:List and approve pending PIM requests')
          _describe 'pim command' pim_cmds
          ;;
        completion)
          local -a shells
          shells=('zsh' 'bash')
          _describe 'shell' shells
          ;;
      esac
      ;;
  esac
}

compdef _eo eo
`

const bashCompletion = `_eo_completion() {
  local cur prev
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"

  case "${prev}" in
    eo)
      COMPREPLY=($(compgen -W "pim version completion help" -- "${cur}"))
      ;;
    pim)
      COMPREPLY=($(compgen -W "approve" -- "${cur}"))
      ;;
    completion)
      COMPREPLY=($(compgen -W "zsh bash" -- "${cur}"))
      ;;
  esac
}

complete -F _eo_completion eo
`

func main() {
	printBanner()

	if err := run(os.Args[1:]); err != nil {
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
	case "pim":
		if len(args) >= 2 && args[1] == "approve" {
			if hasHelpFlag(args[2:]) {
				printPimApproveHelp()
				return nil
			}
			return cmdPimApprove(args[2:])
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

type azureCreds struct {
	subscriptionID string
	userID         string
	accessToken    string
}

func loadCreds(needUser bool) (*azureCreds, error) {
	fmt.Println(dim("→ Authenticating with Azure..."))

	subID, err := pim.GetSubscriptionID()
	if err != nil {
		return nil, fmt.Errorf("getting subscription ID: %w", err)
	}
	token, err := pim.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("getting access token: %w", err)
	}

	creds := &azureCreds{subscriptionID: subID, accessToken: token}

	if needUser {
		userID, err := pim.GetUserID()
		if err != nil {
			return nil, fmt.Errorf("getting user ID: %w", err)
		}
		creds.userID = userID
	}

	return creds, nil
}

func firstPositional(args []string) string {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			return arg
		}
	}
	return ""
}

func cmdPimRequest(args []string) error {
	fmt.Println(bold("→ Activating Contributor role on ") + cyan(pim.SubscriptionName) + bold(" (8h)"))
	fmt.Println()

	creds, err := loadCreds(true)
	if err != nil {
		return err
	}

	justification := "Requesting access to perform necessary tasks."
	if j := firstPositional(args); j != "" {
		justification = j
	}

	err = pim.RequestContributorRole(creds.subscriptionID, creds.userID, creds.accessToken, justification)
	switch {
	case errors.Is(err, pim.ErrRoleAlreadyActive):
		fmt.Println(yellow("⚠ ") + "Contributor role is already active — nothing to do.")
		return nil
	case err != nil:
		return err
	}
	fmt.Println(green("✓ ") + "Contributor role activated.")
	return nil
}

// resolvePrincipals collects the unique principal IDs from the pending
// requests and looks them up via Microsoft Graph in a single call. Returns
// an empty map (never nil) if the lookup fails, so callers can degrade
// gracefully to showing raw IDs.
func resolvePrincipals(pending []pim.ScheduleRequest) map[string]pim.Principal {
	seen := map[string]struct{}{}
	var ids []string
	for _, r := range pending {
		id := r.Properties.PrincipalID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	token, err := pim.GetGraphAccessToken()
	if err != nil {
		return map[string]pim.Principal{}
	}
	result, err := pim.LookupPrincipals(ids, token)
	if err != nil {
		return map[string]pim.Principal{}
	}
	return result
}

func principalLabel(resolved map[string]pim.Principal, id string) string {
	if p, ok := resolved[id]; ok {
		return p.String()
	}
	return id
}

func cmdPimApprove(args []string) error {
	approveAll := false
	for _, arg := range args {
		if arg == "--all" {
			approveAll = true
		}
	}
	justification := "Approved via eo-cli"
	if j := firstPositional(args); j != "" {
		justification = j
	}

	creds, err := loadCreds(false)
	if err != nil {
		return err
	}

	fmt.Println(dim("→ Fetching pending PIM approval requests..."))
	fmt.Println()

	pending, err := pim.ListPendingApprovals(creds.subscriptionID, creds.accessToken)
	if err != nil {
		return fmt.Errorf("listing approvals: %w", err)
	}

	if len(pending) == 0 {
		fmt.Println(green("✓ ") + "No pending approval requests.")
		return nil
	}

	principals := resolvePrincipals(pending)

	fmt.Println(bold(fmt.Sprintf("Found %d pending request(s):", len(pending))))
	fmt.Println()
	for i, r := range pending {
		role := pim.RoleDisplayName(r.Properties.RoleDefinitionID)
		who := principalLabel(principals, r.Properties.PrincipalID)
		fmt.Printf("  %s %s\n", cyan(fmt.Sprintf("[%d]", i+1)), bold(role))
		fmt.Printf("      %s %s\n", dim("Requester:"), who)
		fmt.Printf("      %s    %s\n", dim("Reason:"), r.Properties.Justification)
		fmt.Println()
	}

	toApprove, err := pickApprovals(pending, approveAll)
	if err != nil {
		return err
	}
	if len(toApprove) == 0 {
		fmt.Println(dim("Nothing selected."))
		return nil
	}

	fmt.Println()
	fmt.Println(bold(fmt.Sprintf("Approving %d request(s)...", len(toApprove))))

	var failed int
	for _, r := range toApprove {
		role := pim.RoleDisplayName(r.Properties.RoleDefinitionID)
		who := principalLabel(principals, r.Properties.PrincipalID)
		if err := pim.ApproveScheduleRequest(creds.subscriptionID, creds.accessToken, r, justification); err != nil {
			fmt.Printf("  %s %s for %s\n     %s\n", red("✗"), bold(role), who, dim(err.Error()))
			failed++
			continue
		}
		fmt.Printf("  %s %s for %s\n", green("✓"), bold(role), who)
	}

	if failed > 0 {
		return fmt.Errorf("%d approval(s) failed", failed)
	}
	return nil
}

func pickApprovals(pending []pim.ScheduleRequest, approveAll bool) ([]pim.ScheduleRequest, error) {
	if approveAll {
		return pending, nil
	}

	fmt.Print(bold("› ") + "Enter numbers to approve (space-separated), or " + cyan("all") + ": ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading input: %w", err)
		}
		return nil, nil
	}
	input := strings.TrimSpace(scanner.Text())

	if strings.EqualFold(input, "all") {
		return pending, nil
	}

	var picked []pim.ScheduleRequest
	for _, part := range strings.Fields(input) {
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > len(pending) {
			fmt.Println(yellow("  ⚠ ") + "Invalid selection: " + part)
			continue
		}
		picked = append(picked, pending[n-1])
	}
	return picked, nil
}
