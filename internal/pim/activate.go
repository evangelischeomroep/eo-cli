package pim

import (
	"errors"
	"fmt"
	"net/http"
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
	uuid, err := GenerateUUID()
	if err != nil {
		return err
	}

	body := RequestBody{
		Properties: Properties{
			PrincipalID: userID,
			RoleDefinitionID: fmt.Sprintf(
				"/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s",
				subscriptionID, ContributorRoleID,
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
		armBaseURL, subscriptionID, uuid, scheduleAPIVersion,
	)

	if err := azureRequest(http.MethodPut, url, accessToken, body, nil); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			return ErrRoleAlreadyActive
		}
		return err
	}
	return nil
}
