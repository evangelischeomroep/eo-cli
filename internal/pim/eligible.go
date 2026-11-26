package pim

import (
	"net/http"
	"strings"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
)

type eligibilityInstancesResponse struct {
	Value []struct {
		Properties struct {
			ExpandedProperties struct {
				Scope struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
					Type        string `json:"type"`
				} `json:"scope"`
			} `json:"expandedProperties"`
		} `json:"properties"`
	} `json:"value"`
	NextLink string `json:"nextLink"`
}

// ListEligibleSubscriptions returns the subscriptions in the token's tenant
// where the signed-in user holds a PIM-eligible role at subscription scope.
// `az account list` misses these until one of the roles is active.
func ListEligibleSubscriptions(tenantID, accessToken string) ([]azure.Subscription, error) {
	next := azure.ArmBaseURL + "/providers/Microsoft.Authorization/roleEligibilityScheduleInstances?api-version=" +
		azure.ScheduleAPIVersion + "&$filter=asTarget()"
	var subs []azure.Subscription
	for next != "" {
		var page eligibilityInstancesResponse
		if err := azure.AzureRequest(http.MethodGet, next, accessToken, nil, &page); err != nil {
			return nil, err
		}
		subs = append(subs, eligibleSubscriptions(page, tenantID)...)
		next = page.NextLink
	}
	return subs, nil
}

func eligibleSubscriptions(page eligibilityInstancesResponse, tenantID string) []azure.Subscription {
	var subs []azure.Subscription
	for _, r := range page.Value {
		scope := r.Properties.ExpandedProperties.Scope
		if !strings.EqualFold(scope.Type, "subscription") {
			continue
		}
		parts := strings.Split(scope.ID, "/")
		subs = append(subs, azure.Subscription{
			ID:       parts[len(parts)-1],
			Name:     scope.DisplayName,
			State:    "Enabled",
			TenantID: tenantID,
		})
	}
	return subs
}
