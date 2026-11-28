package azure

import (
	"strings"
	"testing"
)

const accountList = `[
  {"id": "bbb", "name": "EO Studio Digitaal", "state": "Enabled", "isDefault": true},
  {"id": "aaa", "name": "EO AI", "state": "Enabled"},
  {"id": "ccc", "name": "Old", "state": "Disabled"}
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
	if got := strings.Join(names, ","); got != "EO AI,EO Studio Digitaal" {
		t.Errorf("subscriptions = %s, want the enabled ones sorted by name", got)
	}
}

func TestFindSubscription(t *testing.T) {
	subs, _ := parseSubscriptions(accountList)
	for _, test := range []struct{ name, query, wantID, wantErr string }{
		{name: "by name", query: "eo studio digitaal", wantID: "bbb"},
		{name: "by ID", query: "AAA", wantID: "aaa"},
		{name: "unknown", query: "nope", wantErr: "available: EO AI, EO Studio Digitaal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := FindSubscription(subs, test.query)
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
