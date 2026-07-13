package main

import (
	"fmt"

	"github.com/charmbracelet/huh"
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

	deployAll := false
	for _, arg := range args {
		if arg == "--all" || arg == "-a" {
			deployAll = true
		}
	}

	selected := apps
	if !deployAll {
		var err error
		selected, err = pickDeploys(apps)
		if err != nil {
			return err
		}
	}

	fmt.Println(selected)
	return nil
}

func pickDeploys(apps []deploy.FunctionApp) ([]deploy.FunctionApp, error) {
	names := make([]string, len(apps))

	for i, app := range apps {
		names[i] = app.Name
	}

	var selected []string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select Function Apps to deploy").
				Options(huh.NewOptions(names...)...).
				Value(&selected),
		),
	).Run()
	if err != nil {
		return nil, err
	}

	var picked []deploy.FunctionApp
	for _, app := range apps {
		for _, s := range selected {
			if app.Name == s {
				picked = append(picked, app)
			}
		}
	}
	return picked, nil
}
