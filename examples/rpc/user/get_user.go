//nolint
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/rotector/rotector/rpc/user"
)

func main() {
	// Create Twirp client
	client := user.NewUserServiceProtobufClient("http://localhost:8080", &http.Client{})

	// Get user ID from command line
	if len(os.Args) <= 1 {
		fmt.Println("Usage: get_user <user_id>")
		os.Exit(1)
	}
	userID := os.Args[1]

	// Make request
	resp, err := client.GetUser(context.Background(), &user.GetUserRequest{
		UserId: userID,
	})
	if err != nil {
		fmt.Printf("Error getting user: %v\n", err)
		os.Exit(1)
	}

	// Check if user exists
	if resp.GetStatus() == user.UserStatus_USER_STATUS_UNFLAGGED {
		fmt.Printf("User %s: NOT FOUND\n", userID)
		return
	}

	// Pretty print the full response
	jsonBytes, _ := json.MarshalIndent(resp.GetUser(), "", "  ")
	fmt.Println(string(jsonBytes))
}
