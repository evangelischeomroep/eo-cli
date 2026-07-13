package azure

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type azureErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

const (
	ContributorRoleID = "b24988ac-6180-42a0-ab88-20f7382dd24c"
	OwnerRoleID       = "8e3af657-a8ff-443c-a75c-2fe8c4bcb635"
	ReaderRoleID      = "acdd72a7-3385-48ef-bd42-f606fba81ae7"

	SubscriptionName = "EO Studio Digitaal"

	ArmBaseURL         = "https://management.azure.com"
	ScheduleAPIVersion = "2020-10-01"
	ApprovalAPIVersion = "2021-01-01-preview"
)

var ErrRoleAlreadyActive = errors.New("role is already active")

type APIError struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	var parsed azureErrorBody
	if err := json.Unmarshal([]byte(e.Body), &parsed); err == nil && parsed.Error.Message != "" {
		return parsed.Error.Message
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

// AzureRequest performs an authenticated JSON call against Azure REST APIs. Non-2xx
// responses are returned as *APIError so callers can inspect the status code.
func AzureRequest(method, url, accessToken string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		errBody, _ := io.ReadAll(res.Body)
		return &APIError{
			Method:     method,
			URL:        url,
			StatusCode: res.StatusCode,
			Body:       strings.TrimSpace(string(errBody)),
		}
	}

	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			return err
		}
	}
	return nil
}

func azCLI(args ...string) (string, error) {
	out, err := exec.Command("az", args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("az %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("az %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func GetSubscriptionID() (string, error) {
	return azCLI("account", "list", "--query", fmt.Sprintf("[?name=='%s'].id", SubscriptionName), "-o", "tsv")
}

func GetUserID() (string, error) {
	return azCLI("ad", "signed-in-user", "show", "--query", "id", "-o", "tsv")
}

func GetAccessToken() (string, error) {
	return azCLI("account", "get-access-token", "--query", "accessToken", "-o", "tsv")
}

func GetDevOpsAccessToken() (string, error) {
	return azCLI("account", "get-access-token", "--resource", "499b84ac-1321-427f-aa17-267ca6975798", "--query", "accessToken", "-o", "tsv")
}

func GetGraphAccessToken() (string, error) {
	return azCLI("account", "get-access-token", "--resource", "https://graph.microsoft.com", "--query", "accessToken", "-o", "tsv")
}

// GenerateUUID returns a random RFC 4122 v4 UUID.
func GenerateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant RFC 4122
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
