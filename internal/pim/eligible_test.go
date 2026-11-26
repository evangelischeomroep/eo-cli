package pim

import (
	"encoding/json"
	"testing"
)

func TestEligibleSubscriptions(t *testing.T) {
	var page eligibilityInstancesResponse
	err := json.Unmarshal([]byte(`{"value": [
		{"properties": {"expandedProperties": {"scope": {"id": "/subscriptions/aaa", "displayName": "ai-team", "type": "subscription"}}}},
		{"properties": {"expandedProperties": {"scope": {"id": "/subscriptions/bbb/resourceGroups/rg", "displayName": "rg", "type": "resourcegroup"}}}}
	]}`), &page)
	if err != nil {
		t.Fatal(err)
	}
	subs := eligibleSubscriptions(page, "tenant-1")
	if len(subs) != 1 || subs[0].ID != "aaa" || subs[0].Name != "ai-team" || subs[0].TenantID != "tenant-1" {
		t.Fatalf("subscriptions = %+v, want only ai-team (aaa) in tenant-1", subs)
	}
}
