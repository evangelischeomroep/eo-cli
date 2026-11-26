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
	"sort"
	"strings"
	"time"
)

type azureErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Message string `json:"message"`
}

const (
	ContributorRoleID = "b24988ac-6180-42a0-ab88-20f7382dd24c"
	OwnerRoleID       = "8e3af657-a8ff-443c-a75c-2fe8c4bcb635"
	ReaderRoleID      = "acdd72a7-3385-48ef-bd42-f606fba81ae7"

	ArmBaseURL         = "https://management.azure.com"
	ScheduleAPIVersion = "2020-10-01"
	ApprovalAPIVersion = "2021-01-01-preview"
)

var ErrRoleAlreadyActive = errors.New("role is already active")

// ErrLoginRequired means the Azure CLI's sign-in has expired or is missing,
// for example after Conditional Access's sign-in frequency check.
var ErrLoginRequired = errors.New("your Azure sign-in has expired")

type APIError struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	var parsed azureErrorBody
	if err := json.Unmarshal([]byte(e.Body), &parsed); err == nil {
		if parsed.Error.Message != "" {
			return parsed.Error.Message
		}
		if parsed.Message != "" {
			return parsed.Message
		}
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

var httpClient = &http.Client{Timeout: 60 * time.Second}

// AzureRequest performs an authenticated JSON call against Azure REST APIs.
// Non-2xx responses are returned as *APIError so callers can inspect the status code.
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
		var timeoutErr interface{ Timeout() bool }
		if errors.As(err, &timeoutErr) && timeoutErr.Timeout() {
			return fmt.Errorf("Azure API request timed out")
		}
		if cause := errors.Unwrap(err); cause != nil {
			return fmt.Errorf("Azure API request failed: %w", cause)
		}
		return fmt.Errorf("Azure API request failed")
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
			if needsLogin(string(exitErr.Stderr)) {
				return "", fmt.Errorf("%w; run `az login` and try again", ErrLoginRequired)
			}
			return "", fmt.Errorf("az %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("az %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

type Subscription struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	State     string `json:"state"`
	TenantID  string `json:"tenantId"`
	IsDefault bool   `json:"isDefault"`
}

// ListSubscriptions returns the enabled subscriptions the Azure CLI knows for
// the signed-in user, sorted by name. The CLI caches this list at `az login`.
func ListSubscriptions() ([]Subscription, error) {
	out, err := azCLI("account", "list", "-o", "json")
	if err != nil {
		return nil, err
	}
	return parseSubscriptions(out)
}

func parseSubscriptions(raw string) ([]Subscription, error) {
	var all []Subscription
	if err := json.Unmarshal([]byte(raw), &all); err != nil {
		return nil, fmt.Errorf("parsing subscriptions: %w", err)
	}
	var enabled []Subscription
	for _, s := range all {
		if s.State == "Enabled" {
			enabled = append(enabled, s)
		}
	}
	return MergeSubscriptions(enabled), nil
}

// MergeSubscriptions combines lists into one sorted by name, keeping the
// first entry for each ID; the same subscription can be listed once per
// tenant it is reachable from.
func MergeSubscriptions(lists ...[]Subscription) []Subscription {
	seen := map[string]bool{}
	var subs []Subscription
	for _, list := range lists {
		for _, s := range list {
			if seen[strings.ToLower(s.ID)] {
				continue
			}
			seen[strings.ToLower(s.ID)] = true
			subs = append(subs, s)
		}
	}
	sort.SliceStable(subs, func(i, j int) bool {
		return strings.ToLower(subs[i].Name) < strings.ToLower(subs[j].Name)
	})
	return subs
}

// FindSubscription picks the subscription whose ID or name matches query,
// ignoring case.
func FindSubscription(subs []Subscription, query string) (Subscription, error) {
	var byName []Subscription
	for _, s := range subs {
		if strings.EqualFold(s.ID, query) {
			return s, nil
		}
		if strings.EqualFold(s.Name, query) {
			byName = append(byName, s)
		}
	}
	switch len(byName) {
	case 1:
		return byName[0], nil
	case 0:
		names := make([]string, len(subs))
		for i, s := range subs {
			names[i] = s.Name
		}
		if len(names) == 0 {
			return Subscription{}, fmt.Errorf("subscription %q not found", query)
		}
		return Subscription{}, fmt.Errorf("subscription %q not found; available: %s", query, strings.Join(names, ", "))
	default:
		return Subscription{}, fmt.Errorf("more than one subscription is named %q; use its ID instead", query)
	}
}

func GetUserID() (string, error) {
	return azCLI("ad", "signed-in-user", "show", "--query", "id", "-o", "tsv")
}

// GetAccessToken returns an ARM token for the given tenant, or for the
// default tenant when tenantID is empty. It asks by tenant, not subscription,
// because the CLI does not know subscriptions where the user is only
// PIM-eligible.
func GetAccessToken(tenantID string) (string, error) {
	args := []string{"account", "get-access-token", "--query", "accessToken", "-o", "tsv"}
	login := "az login"
	if tenantID != "" {
		args = append(args, "--tenant", tenantID)
		login += " --tenant " + tenantID
	}
	token, err := azCLI(args...)
	if errors.Is(err, ErrLoginRequired) {
		return "", fmt.Errorf("%w; run `%s` and try again", ErrLoginRequired, login)
	}
	return token, err
}

// needsLogin recognises the Azure CLI's "sign in again" errors, which end
// with its advice to run `az login`.
func needsLogin(stderr string) bool {
	return strings.Contains(stderr, "az login") &&
		(strings.Contains(stderr, "AADSTS") || strings.Contains(stderr, "expired"))
}

type AccountInfo struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	State string `json:"state"`
	User  struct {
		Name string `json:"name"`
	} `json:"user"`
}

func GetAccountInfo() (AccountInfo, error) {
	out, err := azCLI("account", "show", "-o", "json")
	if err != nil {
		return AccountInfo{}, err
	}
	var info AccountInfo
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		return AccountInfo{}, fmt.Errorf("parsing account info: %w", err)
	}
	return info, nil
}

func GetSignedInUserDisplayName() (string, error) {
	return azCLI("ad", "signed-in-user", "show", "--query", "displayName", "-o", "tsv")
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
