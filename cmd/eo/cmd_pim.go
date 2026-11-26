package main

import (
	"errors"
	"fmt"
	"time"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
	"github.com/evangelischeomroep/eo-cli/internal/pim"
)

const defaultPimApprovalJustification = "Approved via eo-cli"

func newPimCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pim",
		Short: "Manage Azure PIM activations and approvals",
		Long:  "Manage Azure PIM activations and approvals. With one subscription, eo uses it; with several, it asks which one, unless you pass --subscription.",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	var subscription string
	cmd.PersistentFlags().StringVarP(&subscription, "subscription", "s", "", "Subscription name or ID")
	_ = cmd.RegisterFlagCompletionFunc("subscription", cobra.NoFileCompletions)
	activate := &cobra.Command{
		Use:               "activate [reason]",
		Short:             "Activate the Contributor role for 8 hours",
		Long:              "Activate the Contributor role on a subscription for 8 hours. Azure records the optional reason in the PIM audit log.",
		Example:           "  eo pim activate\n  eo pim activate \"deploying release 2.4\"\n  eo pim activate -s \"EO Studio Digitaal\"",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              func(_ *cobra.Command, args []string) error { return cmdPimRequest(subscription, optionalArg(args)) },
	}
	var approveAll bool
	approve := &cobra.Command{
		Use:               "approve [justification]",
		Short:             "Approve pending PIM requests",
		Long:              fmt.Sprintf("List requests where you are an approver and choose which to approve. The default justification is %q.", defaultPimApprovalJustification),
		Example:           "  eo pim approve\n  eo pim approve --all \"sprint review batch\"",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(_ *cobra.Command, args []string) error {
			return cmdPimApprove(subscription, approveAll, optionalArg(args))
		},
	}
	approve.Flags().BoolVar(&approveAll, "all", false, "Approve all pending requests without prompting")
	cmd.AddCommand(activate, approve, &cobra.Command{
		Use:   "status",
		Short: "Show whether your Contributor role is active and when it expires",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return cmdPimStatus(subscription) },
	})
	return cmd
}

func cmdPimStatus(subscription string) error {
	creds, err := loadCreds(subscription, true)
	if err != nil {
		return err
	}

	expiry, err := pim.GetContributorRoleExpiry(creds.subscription.ID, creds.userID, creds.accessToken)
	if err != nil {
		return err
	}

	remaining := time.Until(expiry)
	if expiry.IsZero() || remaining <= 0 {
		fmt.Println(red("✗ ") + "Contributor role on " + cyan(creds.subscription.Name) + " is not active.")
		return nil
	}

	h := int(remaining.Hours())
	m := int(remaining.Minutes()) % 60
	fmt.Printf("%s Contributor role on %s is active — expires in %s\n", green("✓"), cyan(creds.subscription.Name), bold(fmt.Sprintf("%dh %dm", h, m)))
	return nil
}

func cmdPimRequest(subscription, justification string) error {
	creds, err := loadCreds(subscription, true)
	if err != nil {
		return err
	}
	fmt.Println(bold("→ Activating Contributor role on ") + cyan(creds.subscription.Name) + bold(" (8h)"))
	fmt.Println()

	if justification == "" {
		justification = "Requesting access to perform necessary tasks."
	}

	err = pim.RequestContributorRole(creds.subscription.ID, creds.userID, creds.accessToken, justification)
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

func cmdPimApprove(subscription string, approveAll bool, justification string) error {
	if justification == "" {
		justification = defaultPimApprovalJustification
	}

	creds, err := loadCreds(subscription, false)
	if err != nil {
		return err
	}

	fmt.Println(dim("→ Fetching pending PIM approval requests on " + creds.subscription.Name + "..."))
	fmt.Println()

	pending, err := pim.ListPendingApprovals(creds.subscription.ID, creds.accessToken)
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
		if err := pim.ApproveScheduleRequest(creds.subscription.ID, creds.accessToken, r, justification); err != nil {
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
