package azure

import (
	"strings"
	"testing"
)

const accountList = `[
  {"id": "bbb", "name": "EO Studio Digitaal", "state": "Enabled", "isDefault": true},
  {"id": "aaa", "name": "ai-team", "state": "Enabled"},
  {"id": "ccc", "name": "Old", "state": "Disabled"},
  {"id": "BBB", "name": "EO Studio Digitaal", "state": "Enabled"}
]`

func TestParseSubscriptions(t *testing.T) {
	subs, err := parseSubscriptions(accountList)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range subs {
		names = append(names, s.Name)
	}
	if got := strings.Join(names, ","); got != "ai-team,EO Studio Digitaal" {
		t.Errorf("subscriptions = %s, want enabled, deduplicated and sorted by name", got)
	}
}

func TestFindSubscription(t *testing.T) {
	subs, _ := parseSubscriptions(accountList)
	twins := append(subs, Subscription{ID: "ddd", Name: "ai-team", State: "Enabled"})
	for _, test := range []struct {
		name, query, wantID, wantErr string
		subs                         []Subscription
	}{
		{name: "by name", query: "eo studio digitaal", wantID: "bbb", subs: subs},
		{name: "by ID", query: "AAA", wantID: "aaa", subs: subs},
		{name: "unknown", query: "nope", wantErr: "available: ai-team, EO Studio Digitaal", subs: subs},
		{name: "ambiguous name", query: "ai-team", wantErr: "use its ID", subs: twins},
		{name: "ID beats name", query: "ddd", wantID: "ddd", subs: twins},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := FindSubscription(test.subs, test.query)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, test.wantErr)
				}
				return
			}
			if err != nil || got.ID != test.wantID {
				t.Fatalf("got %q, %v; want %q", got.ID, err, test.wantID)
			}
		})
	}
}

func TestMergeSubscriptions(t *testing.T) {
	known := []Subscription{{ID: "bbb", Name: "EO Studio Digitaal", IsDefault: true}}
	eligible := []Subscription{{ID: "BBB", Name: "EO Studio Digitaal"}, {ID: "aaa", Name: "ai-team"}}
	got := MergeSubscriptions(known, eligible)
	if len(got) != 2 || got[0].ID != "aaa" || !got[1].IsDefault {
		t.Fatalf("merged = %+v, want ai-team and the known EO Studio Digitaal entry", got)
	}
}
