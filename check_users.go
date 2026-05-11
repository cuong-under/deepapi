//go:build tools

package main

import (
	"fmt"
	"log"

	"ds2api/internal/database"
)

func main() {
	db, err := database.Open("ds2api.db")
	if err != nil {
		log.Fatal(err)
	}

	// Count users
	count, err := db.CountUsers()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Total users: %d\n\n", count)

	// Get all users (we'll need to add this method)
	// For now, try to get specific users
	users := []string{"admin", "test", "user", "kohan3"}
	for _, username := range users {
		user, err := db.GetUserByUsername(username)
		if err == nil {
			fmt.Printf("Username: %s\n", user.Username)
			fmt.Printf("Email: %s\n", user.Email)
			fmt.Printf("Role: %s\n", user.Role)
			fmt.Printf("Created: %s\n\n", user.CreatedAt)
		}
	}
}
