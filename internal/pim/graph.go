package pim

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
)

const graphBaseURL = "https://graph.microsoft.com/v1.0"

type Principal struct {
	ID          string
	DisplayName string
	Email       string
	Type        string // "user", "servicePrincipal", "group", ...
}

// String returns "Display Name (email)" if both are known, otherwise falls
// back to whichever field is populated, or the ID.
func (p Principal) String() string {
	switch {
	case p.DisplayName != "" && p.Email != "":
		return fmt.Sprintf("%s (%s)", p.DisplayName, p.Email)
	case p.DisplayName != "":
		return p.DisplayName
	case p.Email != "":
		return p.Email
	default:
		return p.ID
	}
}

func GetGraphAccessToken() (string, error) {
	return azure.GetGraphAccessToken()
}

type graphObject struct {
	ODataType         string `json:"@odata.type"`
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	UserPrincipalName string `json:"userPrincipalName"`
	Mail              string `json:"mail"`
}

// LookupPrincipals resolves a set of principal IDs to display names / emails
// in a single Graph API call. Unresolved IDs are simply absent from the map.
func LookupPrincipals(ids []string, graphToken string) (map[string]Principal, error) {
	if len(ids) == 0 {
		return map[string]Principal{}, nil
	}

	body := map[string]any{
		"ids":   ids,
		"types": []string{"user", "servicePrincipal", "group"},
	}

	var resp struct {
		Value []graphObject `json:"value"`
	}

	url := graphBaseURL + "/directoryObjects/getByIds"
	if err := azure.AzureRequest(http.MethodPost, url, graphToken, body, &resp); err != nil {
		return nil, err
	}

	result := make(map[string]Principal, len(resp.Value))
	for _, obj := range resp.Value {
		p := Principal{
			ID:          obj.ID,
			DisplayName: obj.DisplayName,
			Type:        strings.TrimPrefix(obj.ODataType, "#microsoft.graph."),
		}
		switch {
		case obj.Mail != "":
			p.Email = obj.Mail
		case obj.UserPrincipalName != "":
			p.Email = obj.UserPrincipalName
		}
		result[obj.ID] = p
	}
	return result, nil
}
