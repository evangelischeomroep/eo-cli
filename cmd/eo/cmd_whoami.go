package main

import "fmt"

func cmdWhoami() error {
	user, err := getSignedInUser()
	if err != nil {
		return err
	}

	row := func(label, value string) { fmt.Printf("  %-14s%s\n", label, value) }
	row("Name", bold(user.DisplayName))
	row("Email", user.Email)
	row("Subscription", cyan(user.Subscription))
	return nil
}
