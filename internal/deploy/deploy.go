package deploy

import (
	"fmt"
	"strings"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
)

type FunctionApp struct {
	Name string `json:"name"`
}

type listResponse struct {
	Value []FunctionApp `json:"value"`
}

type Pipeline struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type listPipelinesResponse struct {
	Value []Pipeline `json:"value"`
}

func ListFunctionApps(subscriptionID, accessToken, env string) ([]FunctionApp, error) {
	url := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/rg-%s-webapps/providers/Microsoft.Web/sites?api-version=2022-03-01",
		azure.ArmBaseURL, subscriptionID, env)

	var result listResponse
	if err := azure.AzureRequest("GET", url, accessToken, nil, &result); err != nil {
		return nil, err
	}
	return result.Value, nil
}

const devOpsBaseURL = "https://dev.azure.com/evangelischeomroep/Apps%20and%20Services/_apis"

func ListPipelines(accessToken string) ([]Pipeline, error) {
	url := devOpsBaseURL + "/pipelines?api-version=7.1"

	var result listPipelinesResponse
	if err := azure.AzureRequest("GET", url, accessToken, nil, &result); err != nil {
		return nil, err
	}
	return result.Value, nil
}

type build struct {
	ID int `json:"id"`
}

type buildListResponse struct {
	Value []build `json:"value"`
}

func GetLatestBuild(pipelineID int, accessToken string) (int, error) {
	url := fmt.Sprintf("%s/build/builds?definitions=%d&$top=1&api-version=7.1", devOpsBaseURL, pipelineID)

	var result buildListResponse
	if err := azure.AzureRequest("GET", url, accessToken, nil, &result); err != nil {
		return 0, err
	}
	if len(result.Value) == 0 {
		return 0, fmt.Errorf("no builds found for pipeline %d", pipelineID)
	}
	return result.Value[0].ID, nil
}

type timelineRecord struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Type       string `json:"type"`
	State      string `json:"state"`
}

type timeline struct {
	Records []timelineRecord `json:"records"`
}

func getTimeline(buildID int, accessToken string) ([]timelineRecord, error) {
	url := fmt.Sprintf("%s/build/builds/%d/timeline?api-version=7.1", devOpsBaseURL, buildID)
	var result timeline
	if err := azure.AzureRequest("GET", url, accessToken, nil, &result); err != nil {
		return nil, err
	}
	return result.Records, nil
}

func GetStageIdentifier(buildID int, env, accessToken string) (string, error) {
	records, err := getTimeline(buildID, accessToken)
	if err != nil {
		return "", err
	}
	suffix := "_" + strings.Title(env)
	for _, r := range records {
		if r.Type == "Stage" && strings.HasSuffix(r.Identifier, suffix) {
			return r.Identifier, nil
		}
	}
	return "", fmt.Errorf("no stage found for env %q in build %d", env, buildID)
}

func IsTestStageCompleted(buildID int, accessToken string) (bool, error) {
	records, err := getTimeline(buildID, accessToken)
	if err != nil {
		return false, err
	}
	for _, r := range records {
		if r.Type == "Stage" && strings.HasSuffix(r.Identifier, "_Test") {
			return r.State == "completed", nil
		}
	}
	return false, fmt.Errorf("no test stage found in build %d", buildID)
}

func RunStage(buildID int, stageName, accessToken string) error {
	url := fmt.Sprintf("%s/build/builds/%d/stages/%s?api-version=7.1", devOpsBaseURL, buildID, stageName)

	body := map[string]interface{}{
		"state":              2,
		"forceRetryAllJobs": false,
	}

	return azure.AzureRequest("PATCH", url, accessToken, body, nil)
}
