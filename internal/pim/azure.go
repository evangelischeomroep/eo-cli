package pim

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
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

func GetSubscriptionID() (string, error) {
	cmd, err := exec.Command("az", "account", "list", "--query", "[?name=='EO Studio Digitaal'].id", "-o", "tsv").Output()
	if err != nil {
		return "", err
	}

	subscriptionID := strings.Trim(string(cmd), "\n")
	return subscriptionID, nil
}

func GetUserID() (string, error) {
	cmd, err := exec.Command("az", "ad", "signed-in-user", "show", "--query", "id", "-o", "tsv").Output()
	if err != nil {
		return "", err
	}

	userID := strings.Trim(string(cmd), "\n")
	return userID, nil
}

func GetAccessToken() (string, error) {
	cmd, err := exec.Command("az", "account", "get-access-token", "--query", "accessToken", "-o", "tsv").Output()
	if err != nil {
		return "", err
	}

	accessToken := strings.Trim(string(cmd), "\n")

	return accessToken, nil
}

func GenerateUUID() (string, error) {
	byteSlice := make([]byte, 16)
	_, err := rand.Read(byteSlice)
	if err != nil {
		return "", err
	}

	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", byteSlice[0:4], byteSlice[4:6], byteSlice[6:8], byteSlice[8:10], byteSlice[10:])

	return uuid, nil
}

func RequestContributerRole(subscriptionID, userID, accessToken, justification string) error {
	requestBody := RequestBody{
		Properties: Properties{
			PrincipalID:      userID,
			RoleDefinitionID: fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c", subscriptionID),
			RequestType:      "SelfActivate",
			Justification:    justification,
			ScheduleInfo: ScheduleInfo{
				StartDateTime: nil,
				Expiration: Expiration{
					Type:     "AfterDuration",
					Duration: "PT8H",
				},
			},
		},
	}

	serializedRequestBody, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	uuid, err := GenerateUUID()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("https://management.azure.com/subscriptions/%s/providers/Microsoft.Authorization/roleAssignmentScheduleRequests/%s?api-version=2020-10-01", subscriptionID, uuid), strings.NewReader(string(serializedRequestBody)))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusCreated:
		fmt.Println("Done! Requested Contributor role.")
	case http.StatusConflict:
		fmt.Println("Request failed. Possibly the role is already active.")
	default:
		fmt.Printf("Request failed with status code: %d\n", res.StatusCode)
	}

	return nil
}
