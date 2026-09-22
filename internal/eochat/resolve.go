package eochat

import (
	"fmt"
	"strings"
)

// ResolveModel finds a model by exact ID, exact name, or a unique
// case-insensitive substring match on ID or name.
func ResolveModel(models []Model, query string) (Model, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	for _, m := range models {
		if m.ID == query {
			return m, nil
		}
	}
	for _, m := range models {
		if strings.ToLower(m.Name) == q {
			return m, nil
		}
	}
	var matches []Model
	for _, m := range models {
		if strings.Contains(strings.ToLower(m.ID), q) || strings.Contains(strings.ToLower(m.Name), q) {
			matches = append(matches, m)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return Model{}, fmt.Errorf("no model matches %q — run `eo ask models` to see what is available", query)
	default:
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, m.ID)
		}
		return Model{}, fmt.Errorf("%q matches several models: %s", query, strings.Join(names, ", "))
	}
}

// ResolveKnowledge finds a knowledge base by exact ID, exact name, or a
// unique case-insensitive substring match on the name.
func ResolveKnowledge(items []Knowledge, query string) (Knowledge, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	for _, k := range items {
		if k.ID == query {
			return k, nil
		}
	}
	for _, k := range items {
		if strings.ToLower(k.Name) == q {
			return k, nil
		}
	}
	var matches []Knowledge
	for _, k := range items {
		if strings.Contains(strings.ToLower(k.Name), q) {
			matches = append(matches, k)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return Knowledge{}, fmt.Errorf("no knowledge base matches %q — run `eo ask knowledge` to see what is available", query)
	default:
		names := make([]string, 0, len(matches))
		for _, k := range matches {
			names = append(names, k.Name)
		}
		return Knowledge{}, fmt.Errorf("%q matches several knowledge bases: %s", query, strings.Join(names, ", "))
	}
}
