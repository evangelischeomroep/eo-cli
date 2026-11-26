package main

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
	"github.com/evangelischeomroep/eo-cli/internal/pim"
)

type azureCreds struct {
	subscription azure.Subscription
	userID       string
	accessToken  string
}

// loadCreds resolves the subscription to work on (the --subscription value,
// the only one the user has, or one they pick) and fetches a token for it.
func loadCreds(subscription string, needUser bool) (*azureCreds, error) {
	fmt.Println(dim("→ Authenticating with Azure..."))

	var g errgroup.Group
	var userID string
	if needUser {
		g.Go(func() (err error) {
			userID, err = azure.GetUserID()
			if err != nil {
				return fmt.Errorf("getting user ID: %w", err)
			}
			return nil
		})
	}

	subs, err := listSubscriptions(true)
	if err != nil {
		return nil, fmt.Errorf("listing subscriptions: %w", err)
	}
	sub, err := chooseSubscription(subs, subscription)
	if err != nil {
		return nil, err
	}

	var token string
	g.Go(func() (err error) {
		token, err = azure.GetAccessToken(sub.TenantID)
		if err != nil {
			return fmt.Errorf("getting access token: %w", err)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return &azureCreds{subscription: sub, accessToken: token, userID: userID}, nil
}

// listSubscriptions combines the subscriptions the Azure CLI knows with the
// ones where the user is only PIM-eligible, which the CLI does not list.
// A tenant whose eligibilities cannot be read is skipped, with a warning when
// warn is set.
func listSubscriptions(warn bool) ([]azure.Subscription, error) {
	var known []azure.Subscription
	var tenants []string
	var g errgroup.Group
	g.Go(func() (err error) {
		known, err = azure.ListSubscriptions()
		return err
	})
	g.Go(func() error {
		// Without a tenant list, the default tenant still gets checked.
		ts, err := azure.ListTenants()
		if err != nil || len(ts) == 0 {
			ts = []string{""}
		}
		tenants = ts
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	eligible := make([][]azure.Subscription, len(tenants))
	var wg sync.WaitGroup
	for i, tenant := range tenants {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := azure.GetAccessToken(tenant)
			if err == nil {
				eligible[i], err = pim.ListEligibleSubscriptions(tenant, token)
			}
			if err != nil && warn {
				fmt.Fprintln(os.Stderr, yellow("⚠ ")+dim(fmt.Sprintf("Skipped PIM-eligible subscriptions in tenant %s: %v", tenant, err)))
			}
		}()
	}
	wg.Wait()
	return azure.MergeSubscriptions(append([][]azure.Subscription{known}, eligible...)...), nil
}

func chooseSubscription(subs []azure.Subscription, query string) (azure.Subscription, error) {
	if query != "" {
		return azure.FindSubscription(subs, query)
	}
	switch len(subs) {
	case 0:
		return azure.Subscription{}, errors.New("no Azure subscriptions found; if you were given access recently, run `az login` again")
	case 1:
		return subs[0], nil
	}
	if !stdinIsTerminal() {
		return azure.Subscription{}, fmt.Errorf("you have access to %d subscriptions; choose one with --subscription", len(subs))
	}

	options := make([]huh.Option[int], len(subs))
	selected := 0
	for i, s := range subs {
		options[i] = huh.NewOption(s.Name+" "+dim(s.ID), i)
		if s.IsDefault {
			selected = i
		}
	}
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Select a subscription").
				Options(options...).
				Value(&selected),
		),
	).Run()
	if err != nil {
		return azure.Subscription{}, err
	}
	return subs[selected], nil
}

// completeSubscriptions offers the user's subscription names for --subscription.
func completeSubscriptions(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	subs, err := listSubscriptions(false)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names := make([]cobra.Completion, len(subs))
	for i, s := range subs {
		names[i] = cobra.CompletionWithDesc(s.Name, s.ID)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func optionalArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
