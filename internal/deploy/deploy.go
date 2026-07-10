package deploy

import (
	"fmt"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
)

type FunctionApp struct {
	Name string `json:"name"`
}

type listResponse struct {
	Value []FunctionApp `json:"value"`
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
