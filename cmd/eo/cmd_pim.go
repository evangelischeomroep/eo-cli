package main

import (
	"errors"
	"fmt"
	"time"

	"charm.land/huh/v2"
	"golang.org/x/sync/errgroup"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
	"github.com/evangelischeomroep/eo-cli/internal/pim"
)

func cmdPimStatus() error {
	creds, err := loadCreds(true)
	if err != nil {
		return err
	}

	expiry, err := pim.GetContributorRoleExpiry(creds.subscriptionID, creds.userID, creds.accessToken)
	if err != nil {
		return err
	}

	remaining := time.Until(expiry)
	if expiry.IsZero() || remaining <= 0 {
		fmt.Println(red("✗ ") + "Contributor role is not active.")
		return nil
	}

	h := int(remaining.Hours())
	m := int(remaining.Minutes()) % 60
	fmt.Printf("%s Contributor role is active — expires in %s\n", green("✓"), bold(fmt.Sprintf("%dh %dm", h, m)))
	return nil
}

func cmdPimRequest(args []string) error {
	fmt.Println(bold("→ Activating Contributor role on ") + cyan(azure.SubscriptionName) + bold(" ("+pim.ActivationDurationLabel+")"))
	fmt.Println()

	notify := startNotifyLookup()

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
		// No request was created, so there is nothing for an approver to act on.
		fmt.Println(yellow("⚠ ") + "Contributor role is already active — nothing to do.")
		return nil
	case err != nil:
		return err
	}
	fmt.Println(green("✓ ") + "Contributor role activated.")

	notifySlack(<-notify, justification)
	return nil
}

// pimNotification carries what the Slack message needs, gathered up front.
type pimNotification struct {
	requester  pim.Principal
	webhookURL string
	webhookErr error
}

// startNotifyLookup resolves the requester and the webhook URL in the
// background, so their az round-trips overlap with the activation instead of
// being added to it.
func startNotifyLookup() <-chan pimNotification {
	ch := make(chan pimNotification, 1)
	go func() {
		var n pimNotification
		var g errgroup.Group

		g.Go(func() (err error) {
			n.webhookURL, err = pim.SlackWebhookURL()
			return err
		})
		g.Go(func() error {
			// Best effort: a partial identity still names the requester.
			user, _ := getSignedInUser()
			n.requester = pim.Principal{DisplayName: user.DisplayName, Email: user.Email}
			return nil
		})

		n.webhookErr = g.Wait()
		ch <- n
	}()
	return ch
}

// notifySlack tells the team a request is waiting. The role is already
// requested by the time this runs, so nothing here may fail the command.
func notifySlack(n pimNotification, justification string) {
	if n.webhookErr != nil {
		fmt.Println(dim("  Slack notification skipped: " + n.webhookErr.Error()))
		return
	}
	if err := pim.NotifyActivationRequested(n.webhookURL, n.requester, justification); err != nil {
		fmt.Println(yellow("⚠ ") + dim("Slack notification failed: "+err.Error()))
		return
	}
	fmt.Println(dim("  Slack notified — approvers can run ") + cyan("eo pim approve"))
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
	approveAll := hasFlag(args, "--all")
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

	toApprove, err := pickApprovals(pending, principals, approveAll)
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

func pickApprovals(pending []pim.ScheduleRequest, principals map[string]pim.Principal, approveAll bool) ([]pim.ScheduleRequest, error) {
	if approveAll {
		return pending, nil
	}

	options := make([]huh.Option[int], len(pending))
	for i, r := range pending {
		role := pim.RoleDisplayName(r.Properties.RoleDefinitionID)
		who := principalLabel(principals, r.Properties.PrincipalID)
		label := fmt.Sprintf("%s — %s", role, who)
		if r.Properties.Justification != "" {
			label += "\n  " + dim(r.Properties.Justification)
		}
		options[i] = huh.NewOption(label, i)
	}

	var selected []int
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select requests to approve").
				Options(options...).
				Value(&selected),
		),
	).Run()
	if err != nil {
		return nil, err
	}

	picked := make([]pim.ScheduleRequest, 0, len(selected))
	for _, i := range selected {
		picked = append(picked, pending[i])
	}
	return picked, nil
}
