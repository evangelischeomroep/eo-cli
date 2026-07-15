package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/evangelischeomroep/eo-cli/internal/azure"
	"github.com/evangelischeomroep/eo-cli/internal/deploy"
)

//go:embed pipeline-map.json
var pipelineMapData []byte

func loadPipelineMap() (map[string]string, error) {
	var m map[string]string
	if err := json.Unmarshal(pipelineMapData, &m); err != nil {
		return nil, fmt.Errorf("pipeline-map.json is invalid: %w", err)
	}
	return m, nil
}

func pipelineName(baseName string, overrides map[string]string) string {
	if overrides != nil {
		if override, ok := overrides[baseName]; ok {
			return override + "_release"
		}
	}
	return baseName + "_release"
}

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

	pipelineOverrides, err := loadPipelineMap()
	if err != nil {
		return err
	}

	if env == "prod" {
		appNames := make([]string, len(selected))
		for i, app := range selected {
			appNames[i] = "  • " + app.Name
		}
		var confirmed bool
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Deploy %d app(s) to production?", len(selected))).
					Description(strings.Join(appNames, "\n")).
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

	watch := hasFlag(args, "--status", "-s")
	var watching []watchedDeploy
	var failed int

	for _, app := range selected {
		baseName := strings.TrimPrefix(app.Name, "app-"+env+"-")
		pname := pipelineName(baseName, pipelineOverrides)
		p, ok := pipelineByName[pname]
		if !ok {
			fmt.Printf("  %s %s — no pipeline found (expected %q)\n", red("✗"), app.Name, pname)
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
			if watch {
				stageName, err := deploy.GetStageIdentifier(buildID, env, devOpsToken)
				if err != nil {
					fmt.Printf("     %s %s — skipping status watch: %s\n", dim("⚠"), app.Name, dim(err.Error()))
				} else {
					watching = append(watching, watchedDeploy{app.Name, buildID, stageName})
				}
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
			if watch {
				watching = append(watching, watchedDeploy{app.Name, buildID, stageName})
			}
		}
		fmt.Printf("  %s %s\n", green("✓"), app.Name)
	}

	if watch && len(watching) > 0 {
		if err := watchDeployments(watching, devOpsToken); err != nil {
			return err
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d deployment(s) failed", failed)
	}
	return nil
}

type watchedDeploy struct {
	appName   string
	buildID   int
	stageName string
}

func watchDeployments(deployments []watchedDeploy, devOpsToken string) error {
	fmt.Println(dim("\nWatching deployments..."))

	lastState := make(map[string]string, len(deployments))
	remaining := len(deployments)
	consecutiveErrors := 0
	var stageFailed int

	printState := func(d watchedDeploy, state, result string) {
		switch {
		case state == deploy.StageStateCompleted && result == deploy.StageResultSucceeded:
			fmt.Printf("  %s %s\n", green("✓"), d.appName)
		case state == deploy.StageStateCompleted:
			fmt.Printf("  %s %s — %s\n", red("✗"), d.appName, result)
		case state == deploy.StageStateInProgress:
			fmt.Printf("  %s %s — running\n", dim("⏳"), d.appName)
		default:
			fmt.Printf("  %s %s — pending\n", dim("·"), d.appName)
		}
	}

	for _, d := range deployments {
		state, result, err := deploy.GetStageStatus(d.buildID, d.stageName, devOpsToken)
		if err != nil {
			continue
		}
		lastState[d.appName] = state
		if state == deploy.StageStateCompleted {
			remaining--
			if result != deploy.StageResultSucceeded {
				stageFailed++
			}
		}
		printState(d, state, result)
	}

	for remaining > 0 {
		time.Sleep(10 * time.Second)

		intervalHadError := false
		for _, d := range deployments {
			if lastState[d.appName] == deploy.StageStateCompleted {
				continue
			}
			state, result, err := deploy.GetStageStatus(d.buildID, d.stageName, devOpsToken)
			if err != nil {
				intervalHadError = true
				continue
			}
			if state == lastState[d.appName] {
				continue
			}
			lastState[d.appName] = state
			printState(d, state, result)
			if state == deploy.StageStateCompleted {
				remaining--
				if result != deploy.StageResultSucceeded {
					stageFailed++
				}
			}
		}
		if intervalHadError {
			consecutiveErrors++
			if consecutiveErrors >= 5 {
				fmt.Fprintf(os.Stderr, "  %s polling failed repeatedly — giving up\n", red("✗"))
				return fmt.Errorf("polling failed repeatedly")
			}
		} else {
			consecutiveErrors = 0
		}
	}

	if stageFailed > 0 {
		return fmt.Errorf("%d deployment(s) did not succeed", stageFailed)
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
