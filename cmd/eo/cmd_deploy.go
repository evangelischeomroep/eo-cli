package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/evangelischeomroep/eo-cli/internal/azure"
	"github.com/evangelischeomroep/eo-cli/internal/deploy"
)

func cmdDeploy(args []string) error {
	env := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		env = args[0]
	}
	if env == "" {
		if hasFlag(args, "--all", "-a") {
			return fmt.Errorf("specify an environment: eo deploy test --all or eo deploy prod --all")
		}
		env = "test"
	}
	if env != "test" && env != "prod" {
		return fmt.Errorf("unknown environment %q — use test or prod", env)
	}

	creds, err := loadCreds(false)
	if err != nil {
		return err
	}

	apps, err := deploy.ListFunctionApps(creds.subscriptionID, creds.accessToken, env)
	if err != nil {
		return err
	}

	selected := apps
	if !hasFlag(args, "--all", "-a") {
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

	if env == "prod" {
		var confirmed bool
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Deploy %d app(s) to production?", len(selected))).
					Value(&confirmed),
			),
		).Run()
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println(dim("Cancelled."))
			return nil
		}
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
		if env == "prod" {
			ok, err := deploy.IsTestStageCompleted(buildID, devOpsToken)
			if err != nil {
				fmt.Printf("  %s %s — %s\n", red("✗"), app.Name, dim(err.Error()))
				failed++
				continue
			}
			if !ok {
				fmt.Printf("  %s %s — test stage not completed yet\n", red("✗"), app.Name)
				failed++
				continue
			}
			if err := approveProd(buildID, app.Name, devOpsToken); err != nil {
				fmt.Printf("  %s %s — %s\n", red("✗"), app.Name, dim(err.Error()))
				failed++
				continue
			}
		} else {
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
		}
		fmt.Printf("  %s %s\n", green("✓"), app.Name)
	}

	if failed > 0 {
		return fmt.Errorf("%d deployment(s) failed", failed)
	}
	return nil
}

func approveProd(buildID int, appName, devOpsToken string) error {
	approvals, err := deploy.GetPendingApprovals(buildID, devOpsToken)
	if err != nil {
		return fmt.Errorf("could not fetch approvals: %w", err)
	}
	if len(approvals) == 0 {
		return fmt.Errorf("no pending approval found")
	}
	if len(approvals) > 1 {
		return fmt.Errorf("expected 1 pending approval but found %d — aborting to be safe", len(approvals))
	}
	fmt.Printf("     %s approving build %d for %s\n", dim("→"), buildID, bold(appName))
	if err := deploy.ApproveDeployment(approvals[0].ID, devOpsToken); err != nil {
		return fmt.Errorf("approval failed: %w", err)
	}
	return nil
}

func pickDeploys(apps []deploy.FunctionApp) ([]deploy.FunctionApp, error) {
	options := make([]huh.Option[int], len(apps))
	for i, app := range apps {
		options[i] = huh.NewOption(app.Name, i)
	}

	var selected []int
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select Function Apps to deploy").
				Options(options...).
				Value(&selected),
		),
	).Run()
	if err != nil {
		return nil, err
	}

	picked := make([]deploy.FunctionApp, 0, len(selected))
	for _, i := range selected {
		picked = append(picked, apps[i])
	}
	return picked, nil
}
