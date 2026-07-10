package main

import (
	"fmt"

	"github.com/evangelischeomroep/eo-cli/internal/deploy"
)

func cmdDeploy(args []string) error {
	creds, err := loadCreds(false)
	if err != nil {
		return err
	}
	env := "test"
	if len(args) > 0 {
		env = args[0]
	}
	apps, err := deploy.ListFunctionApps(creds.subscriptionID, creds.accessToken, env)
	if err != nil {
		return err
	}

	for _, app := range apps {
		fmt.Println(app.Name)
	}

	return nil
}
