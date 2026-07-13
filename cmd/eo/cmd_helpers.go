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
