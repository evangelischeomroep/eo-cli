package main

import (
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
)

type azureCreds struct {
	subscriptionID string
	userID         string
	accessToken    string
}

func loadCreds(needUser bool) (*azureCreds, error) {
	fmt.Println(dim("→ Authenticating with Azure..."))

	var subID, token, userID string
	var g errgroup.Group

	g.Go(func() (err error) {
		subID, err = azure.GetSubscriptionID()
		if err != nil {
			return fmt.Errorf("getting subscription ID: %w", err)
		}
		return nil
	})
	g.Go(func() (err error) {
		token, err = azure.GetAccessToken()
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
	return &azureCreds{subscriptionID: subID, accessToken: token, userID: userID}, nil
}

type signedInUser struct {
	DisplayName  string
	Email        string
	Subscription string
}

// getSignedInUser resolves the logged-in Azure identity. On error the returned
// struct still holds whatever did resolve, so callers that only decorate output
// can degrade instead of giving up.
func getSignedInUser() (signedInUser, error) {
	var user signedInUser
	var g errgroup.Group

	g.Go(func() error {
		account, err := azure.GetAccountInfo()
		if err != nil {
			return fmt.Errorf("getting account info: %w", err)
		}
		user.Email = account.User.Name
		user.Subscription = account.Name
		return nil
	})
	g.Go(func() (err error) {
		user.DisplayName, err = azure.GetSignedInUserDisplayName()
		if err != nil {
			return fmt.Errorf("getting display name: %w", err)
		}
		return nil
	})

	return user, g.Wait()
}

func hasFlag(args []string, flags ...string) bool {
	for _, arg := range args {
		for _, f := range flags {
			if arg == f {
				return true
			}
		}
	}
	return false
}

func firstPositional(args []string) string {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
	}
	return ""
}
