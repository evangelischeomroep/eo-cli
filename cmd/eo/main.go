package main

import (
	"fmt"
	"os"

	"github.com/evangelischeomroep/eo-cli/internal/pim"
)

func main() {
	fmt.Println(`
    ______ ____     ______ __     ____ 
   / ____// __ \   / ____// /    /  _/ 
  / __/  / / / /  / /    / /     / /   
 / /___ / /_/ /  / /___ / /___ _/ /    
/_____/ \____/   \____//_____//___/    
                                       
	`)

	if len(os.Args) < 2 {
		fmt.Println("Usage: eo <command> [arguments]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "pim":

		fmt.Println("Request Contributor role on studiodigitaal (8h)...")
		subscriptionID, err := pim.GetSubscriptionID()
		if err != nil {
			fmt.Println("Error getting subscription ID:", err)
			os.Exit(1)
		}

		userID, err := pim.GetUserID()
		if err != nil {
			fmt.Println("Error getting user ID:", err)
			os.Exit(1)
		}

		accessToken, err := pim.GetAccessToken()
		if err != nil {
			fmt.Println("Error getting access token:", err)
			os.Exit(1)
		}

		var justification string
		if len(os.Args) > 2 {
			justification = os.Args[2]
		} else {
			justification = "Requesting access to perform necessary tasks."
		}

		err = pim.RequestContributerRole(subscriptionID, userID, accessToken, justification)
		if err != nil {
			fmt.Println("Error requesting contributer role:", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Unknown command:", command)
		os.Exit(1)
	}
}
