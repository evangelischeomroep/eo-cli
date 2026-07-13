package main

import (
	"fmt"
	"strings"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
)

type azureCreds struct {
	subscriptionID string
	userID         string
	accessToken    string
}

func loadCreds(needUser bool) (*azureCreds, error) {
	fmt.Println(dim("→ Authenticating with Azure..."))

	subID, err := azure.GetSubscriptionID()
	if err != nil {
		return nil, fmt.Errorf("getting subscription ID: %w", err)
	}
	token, err := azure.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("getting access token: %w", err)
	}

	creds := &azureCreds{subscriptionID: subID, accessToken: token}

	if needUser {
		userID, err := azure.GetUserID()
		if err != nil {
			return nil, fmt.Errorf("getting user ID: %w", err)
		}
		creds.userID = userID
	}

	return creds, nil
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
		if !strings.HasPrefix(arg, "--") {
			return arg
		}
	}
	return ""
}
