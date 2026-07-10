package pim

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
)

var wellKnownRoles = map[string]string{
	azure.ContributorRoleID: "Contributor",
	azure.OwnerRoleID:       "Owner",
	azure.ReaderRoleID:      "Reader",
}

type ScheduleRequestProperties struct {
	PrincipalID      string `json:"principalId"`
	RoleDefinitionID string `json:"roleDefinitionId"`
	Status           string `json:"status"`
	Justification    string `json:"justification"`
	ApprovalID       string `json:"approvalId"`
}

type ScheduleRequest struct {
	ID         string                    `json:"id"`
	Name       string                    `json:"name"`
	Properties ScheduleRequestProperties `json:"properties"`
}

type ScheduleRequestsResponse struct {
	Value []ScheduleRequest `json:"value"`
}

type ApprovalStageProperties struct {
	Status       string `json:"status"`
	ReviewResult string `json:"reviewResult"`
}

type ApprovalStage struct {
	ID         string                  `json:"id"`
	Properties ApprovalStageProperties `json:"properties"`
}

type ApprovalProperties struct {
	Status string          `json:"status"`
	Stages []ApprovalStage `json:"stages"`
}

type Approval struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Properties ApprovalProperties `json:"properties"`
}

func RoleDisplayName(roleDefinitionID string) string {
	parts := strings.Split(roleDefinitionID, "/")
	guid := parts[len(parts)-1]
	if name, ok := wellKnownRoles[guid]; ok {
		return name
	}
	return guid
}

// ListPendingApprovals returns role activation requests awaiting the current
// user's approval.
func ListPendingApprovals(subscriptionID, accessToken string) ([]ScheduleRequest, error) {
	url := fmt.Sprintf(
		"%s/subscriptions/%s/providers/Microsoft.Authorization/roleAssignmentScheduleRequests?api-version=%s&$filter=asApprover()",
		azure.ArmBaseURL, subscriptionID, azure.ScheduleAPIVersion,
	)

	var response ScheduleRequestsResponse
	if err := azure.AzureRequest(http.MethodGet, url, accessToken, nil, &response); err != nil {
		return nil, err
	}

	var pending []ScheduleRequest
	for _, r := range response.Value {
		if r.Properties.Status == "PendingApproval" {
			pending = append(pending, r)
		}
	}
	return pending, nil
}

// ApproveScheduleRequest looks up the active approval stage for the given
// request and marks it as Approved.
func ApproveScheduleRequest(subscriptionID, accessToken string, request ScheduleRequest, justification string) error {
	if request.Properties.ApprovalID == "" {
		return fmt.Errorf("no approval ID found for request %s", request.Name)
	}

	// ApprovalID can be a full resource ID; keep only the trailing GUID.
	parts := strings.Split(request.Properties.ApprovalID, "/")
	approvalName := parts[len(parts)-1]

	approvalURL := fmt.Sprintf(
		"%s/subscriptions/%s/providers/Microsoft.Authorization/roleAssignmentApprovals/%s?api-version=%s",
		azure.ArmBaseURL, subscriptionID, approvalName, azure.ApprovalAPIVersion,
	)

	var approval Approval
	if err := azure.AzureRequest(http.MethodGet, approvalURL, accessToken, nil, &approval); err != nil {
		return fmt.Errorf("fetching approval: %w", err)
	}

	var activeStageName string
	for _, stage := range approval.Properties.Stages {
		if stage.Properties.Status == "InProgress" {
			// stage.ID can be a full resource path; keep only the trailing GUID.
			p := strings.Split(stage.ID, "/")
			activeStageName = p[len(p)-1]
			break
		}
	}
	if activeStageName == "" {
		return fmt.Errorf("no active stage found for approval %s", approvalName)
	}

	stageURL := fmt.Sprintf(
		"%s/subscriptions/%s/providers/Microsoft.Authorization/roleAssignmentApprovals/%s/stages/%s?api-version=%s",
		azure.ArmBaseURL, subscriptionID, approvalName, activeStageName, azure.ApprovalAPIVersion,
	)

	body := map[string]any{
		"properties": map[string]string{
			"reviewResult":  "Approve",
			"justification": justification,
		},
	}

	return azure.AzureRequest(http.MethodPatch, stageURL, accessToken, body, nil)
}
