package main

import (
	"errors"
	"fmt"

	"charm.land/huh/v2"
	"golang.org/x/sync/errgroup"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
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

	subs, err := azure.ListSubscriptions()
	if err != nil {
		return nil, fmt.Errorf("listing subscriptions: %w", err)
	}
	sub, err := chooseSubscription(subs, subscription)
	if err != nil {
		return nil, err
	}

	var token, userID string
	var g errgroup.Group
	g.Go(func() (err error) {
		token, err = azure.GetAccessToken(sub.ID)
		if err != nil {
			return fmt.Errorf("getting access token: %w", err)
		}
		return nil
	})
	if needUser {
		g.Go(func() (err error) {
			userID, err = azure.GetUserID()
			if err != nil {
				return fmt.Errorf("getting user ID: %w", err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return &azureCreds{subscription: sub, accessToken: token, userID: userID}, nil
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

func optionalArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
