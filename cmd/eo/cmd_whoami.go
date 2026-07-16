package main

import (
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
)

func cmdWhoami() error {
	var account azure.AccountInfo
	var displayName string

	var g errgroup.Group
	g.Go(func() (err error) {
		account, err = azure.GetAccountInfo()
		if err != nil {
			return fmt.Errorf("getting account info: %w", err)
		}
		return nil
	})
	g.Go(func() (err error) {
		displayName, err = azure.GetSignedInUserDisplayName()
		if err != nil {
			return fmt.Errorf("getting display name: %w", err)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return err
	}

	row := func(label, value string) { fmt.Printf("  %-14s%s\n", label, value) }
	row("Name", bold(displayName))
	row("Email", account.User.Name)
	row("Subscription", cyan(account.Name))
	return nil
}
