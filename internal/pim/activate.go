package pim

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
)

type RequestBody struct {
	Properties Properties `json:"properties"`
}

type Properties struct {
	PrincipalID      string `json:"principalId"`
	RoleDefinitionID string `json:"roleDefinitionId"`
	RequestType      string `json:"requestType"`
	Justification    string `json:"justification"`
	ScheduleInfo     `json:"scheduleInfo"`
}

type ScheduleInfo struct {
	StartDateTime *string `json:"startDateTime"`
	Expiration    `json:"expiration"`
}

type Expiration struct {
	Type     string `json:"type"`
	Duration string `json:"duration"`
}

// RequestContributorRole activates the Contributor role on the given
// subscription for 8 hours. Returns ErrRoleAlreadyActive if the role is
// already active.
func RequestContributorRole(subscriptionID, userID, accessToken, justification string) error {
	uuid, err := azure.GenerateUUID()
	if err != nil {
		return err
	}

	body := RequestBody{
		Properties: Properties{
			PrincipalID: userID,
			RoleDefinitionID: fmt.Sprintf(
				"/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s",
				subscriptionID, azure.ContributorRoleID,
			),
			RequestType:   "SelfActivate",
			Justification: justification,
			ScheduleInfo: ScheduleInfo{
				Expiration: Expiration{
					Type:     "AfterDuration",
					Duration: "PT8H",
				},
			},
		},
	}

	url := fmt.Sprintf(
		"%s/subscriptions/%s/providers/Microsoft.Authorization/roleAssignmentScheduleRequests/%s?api-version=%s",
		azure.ArmBaseURL, subscriptionID, uuid, azure.ScheduleAPIVersion,
	)

	if err := azure.AzureRequest(http.MethodPut, url, accessToken, body, nil); err != nil {
		var apiErr *azure.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			return azure.ErrRoleAlreadyActive
		}
		return err
	}
	return nil
}

type roleAssignmentScheduleInstance struct {
	Properties struct {
		PrincipalID      string `json:"principalId"`
		RoleDefinitionID string `json:"roleDefinitionId"`
		EndDateTime      string `json:"endDateTime"`
		Status           string `json:"status"`
	} `json:"properties"`
}

type roleAssignmentScheduleInstancesResponse struct {
	Value    []roleAssignmentScheduleInstance `json:"value"`
	NextLink string                           `json:"nextLink"`
}

// GetContributorRoleExpiry returns the expiry time of the active Contributor role
// assignment for the given user. Returns a zero time.Time when the role is not active.
func GetContributorRoleExpiry(subscriptionID, userID, accessToken string) (time.Time, error) {
	next := fmt.Sprintf(
		"%s/subscriptions/%s/providers/Microsoft.Authorization/roleAssignmentScheduleInstances?api-version=%s&$filter=atScope()",
		azure.ArmBaseURL, subscriptionID, azure.ScheduleAPIVersion,
	)
	suffix := "/roleDefinitions/" + azure.ContributorRoleID

	for next != "" {
		var page roleAssignmentScheduleInstancesResponse
		if err := azure.AzureRequest(http.MethodGet, next, accessToken, nil, &page); err != nil {
			return time.Time{}, err
		}
		for _, r := range page.Value {
			if r.Properties.PrincipalID == userID &&
				r.Properties.Status == "Active" &&
				strings.HasSuffix(r.Properties.RoleDefinitionID, suffix) {
				t, err := time.Parse(time.RFC3339Nano, r.Properties.EndDateTime)
				if err != nil {
					return time.Time{}, fmt.Errorf("parsing expiry time: %w", err)
				}
				return t, nil
			}
		}
		next = page.NextLink
	}
	return time.Time{}, nil
}
