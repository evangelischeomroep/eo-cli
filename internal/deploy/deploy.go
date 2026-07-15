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

const (
	StageStatePending    = "pending"
	StageStateInProgress = "inProgress"
	StageStateCompleted  = "completed"
	StageResultSucceeded = "succeeded"
)

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
	url := fmt.Sprintf("%s/build/builds?definitions=%d&$top=1&queryOrder=queueTimeDescending&api-version=7.1", devOpsBaseURL, pipelineID)

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
	Result     string `json:"result"`
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

var envSuffix = map[string]string{
	"test": "_Test",
	"prod": "_Production",
}

func GetStageIdentifier(buildID int, env, accessToken string) (string, error) {
	records, err := getTimeline(buildID, accessToken)
	if err != nil {
		return "", err
	}
	suffix, ok := envSuffix[env]
	if !ok {
		return "", fmt.Errorf("unknown env %q", env)
	}
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
	suffix := envSuffix["test"]
	for _, r := range records {
		if r.Type == "Stage" && strings.HasSuffix(r.Identifier, suffix) {
			return r.State == "completed", nil
		}
	}
	return false, fmt.Errorf("no test stage found in build %d", buildID)
}

type Approval struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Pipeline struct {
		Owner struct {
			ID int `json:"id"`
		} `json:"owner"`
	} `json:"pipeline"`
}

type approvalListResponse struct {
	Value []Approval `json:"value"`
}

func GetPendingApprovals(buildID int, accessToken string) ([]Approval, error) {
	url := fmt.Sprintf("%s/pipelines/approvals?api-version=7.1-preview.1", devOpsBaseURL)

	var result approvalListResponse
	if err := azure.AzureRequest("GET", url, accessToken, nil, &result); err != nil {
		return nil, err
	}

	var pending []Approval
	for _, a := range result.Value {
		if a.Status == "pending" && a.Pipeline.Owner.ID == buildID {
			pending = append(pending, a)
		}
	}
	return pending, nil
}

func ApproveDeployment(approvalID, accessToken string) error {
	url := fmt.Sprintf("%s/pipelines/approvals?api-version=7.1-preview.1", devOpsBaseURL)

	body := []map[string]interface{}{
		{
			"approvalId": approvalID,
			"status":     "approved",
			"comment":    "Approved via eo-cli",
		},
	}

	return azure.AzureRequest("PATCH", url, accessToken, body, nil)
}

func GetStageStatus(buildID int, stageIdentifier, accessToken string) (state, result string, err error) {
	records, err := getTimeline(buildID, accessToken)
	if err != nil {
		return "", "", err
	}
	for _, r := range records {
		if r.Type == "Stage" && r.Identifier == stageIdentifier {
			return r.State, r.Result, nil
		}
	}
	return "", "", fmt.Errorf("stage %q not found in build %d", stageIdentifier, buildID)
}

func RunStage(buildID int, stageName, accessToken string) error {
	url := fmt.Sprintf("%s/build/builds/%d/stages/%s?api-version=7.1", devOpsBaseURL, buildID, stageName)

	body := map[string]interface{}{
		"state":              2,
		"forceRetryAllJobs": false,
	}

	return azure.AzureRequest("PATCH", url, accessToken, body, nil)
}
