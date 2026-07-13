package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/evangelischeomroep/eo-cli/internal/azure"
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

	devOpsToken, err := azure.GetDevOpsAccessToken()
	if err != nil {
		return fmt.Errorf("getting DevOps token: %w", err)
	}

	pipelines, err := deploy.ListPipelines(devOpsToken)
	if err != nil {
		return fmt.Errorf("listing pipelines: %w", err)
	}

	pipelineByName := make(map[string]deploy.Pipeline)
	for _, p := range pipelines {
		pipelineByName[p.Name] = p
	}

	var failed int
	for _, app := range selected {
		baseName := strings.TrimPrefix(app.Name, "app-"+env+"-")
		pipelineName := baseName + "_release"
		p, ok := pipelineByName[pipelineName]
		if !ok {
			fmt.Printf("  %s %s — no pipeline found\n", red("✗"), app.Name)
			failed++
			continue
		}
		buildID, err := deploy.GetLatestBuild(p.ID, devOpsToken)
		if err != nil {
			fmt.Printf("  %s %s\n     %s\n", red("✗"), app.Name, dim(err.Error()))
			failed++
			continue
		}
		stageName, err := deploy.GetStageIdentifier(buildID, env, devOpsToken)
		if err != nil {
			fmt.Printf("  %s %s — %s\n", red("✗"), app.Name, dim(err.Error()))
			failed++
			continue
		}
if err := deploy.RunStage(buildID, stageName, devOpsToken); err != nil {
			fmt.Printf("  %s %s — stage failed\n     %s\n", red("✗"), app.Name, dim(err.Error()))
			failed++
			continue
		}
		fmt.Printf("  %s %s\n", green("✓"), app.Name)
	}

	if failed > 0 {
		return fmt.Errorf("%d deployment(s) failed", failed)
	}
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
