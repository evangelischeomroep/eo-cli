package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
	"github.com/evangelischeomroep/eo-cli/internal/pim"
)

func cmdPimRequest(args []string) error {
	fmt.Println(bold("→ Activating Contributor role on ") + cyan(azure.SubscriptionName) + bold(" (8h)"))
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
	case errors.Is(err, azure.ErrRoleAlreadyActive):
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
