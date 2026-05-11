//go:build tools

package main

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "ds2api.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Check users and their accounts
	rows, err := db.Query(`
		SELECT u.id, u.username, u.role, COUNT(a.id) as accounts_count
		FROM users u
		LEFT JOIN user_accounts a ON u.id = a.user_id
		GROUP BY u.id
	`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	fmt.Println("Users and their accounts:")
	fmt.Println("ID | Username | Role | Accounts")
	fmt.Println("---|----------|------|----------")
	for rows.Next() {
		var id int64
		var username, role string
		var count int
		rows.Scan(&id, &username, &role, &count)
		fmt.Printf("%d | %s | %s | %d\n", id, username, role, count)
	}

	// Check all accounts with more details
	fmt.Println("\nAll accounts (detailed):")
	rows2, err := db.Query(`SELECT id, user_id, name, email, password FROM user_accounts`)
	if err != nil {
		panic(err)
	}
	defer rows2.Close()

	fmt.Println("ID | UserID | Name | Email | HasPassword")
	fmt.Println("---|--------|------|-------|-------------")
	for rows2.Next() {
		var id, userID int64
		var name, email, password sql.NullString
		rows2.Scan(&id, &userID, &name, &email, &password)
		hasPass := "No"
		if password.Valid && password.String != "" {
			hasPass = "Yes"
		}
		fmt.Printf("%d | %d | %s | %s | %s\n", id, userID, name.String, email.String, hasPass)
	}

	// Check API keys
	fmt.Println("\nAPI Keys:")
	rows3, err := db.Query(`SELECT id, user_id, api_key, name FROM user_api_keys`)
	if err != nil {
		fmt.Println("Error querying user_api_keys:", err)
		fmt.Println("Table might not exist yet")
	} else {
		defer rows3.Close()

		fmt.Println("ID | UserID | Key | Name")
		fmt.Println("---|--------|-----|------")
		for rows3.Next() {
			var id, userID int64
			var apiKey string
			var name sql.NullString
			rows3.Scan(&id, &userID, &apiKey, &name)
			keyPreview := apiKey
			if len(apiKey) > 20 {
				keyPreview = apiKey[:20] + "..."
			}
			fmt.Printf("%d | %d | %s | %s\n", id, userID, keyPreview, name.String)
		}
	}

	// Check all tables
	fmt.Println("\nAll tables in database:")
	rows4, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	if err != nil {
		panic(err)
	}
	defer rows4.Close()

	for rows4.Next() {
		var tableName string
		rows4.Scan(&tableName)
		fmt.Println("-", tableName)
	}
}
