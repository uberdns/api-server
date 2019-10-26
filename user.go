package main

import (
	"database/sql"
	"fmt"
	"log"
)

// User -- struct for storing information regarding a user performing a query
type User struct {
	ID    int
	Name  string
	Admin bool
	Staff bool
}

func (u *User) LookupFromAPIKey(apiKey string) {
	var isAdmin int
	var isStaff int

	query := "SELECT auth_user.id, auth_user.username, auth_user.is_superuser, auth_user.is_staff FROM auth_user INNER JOIN authtoken_token ON auth_user.id = authtoken_token.user_id WHERE authtoken_token.key = ?"

	dq, err := dbConn.Prepare(query)

	if err != nil {
		log.Fatal(err)
	}

	defer dq.Close()

	err = dq.QueryRow(apiKey).Scan(&u.ID, &u.Name, &isAdmin, &isStaff)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No user found by that API Key")
			return
		}
		log.Fatal(err)
	}

	if isAdmin == 1 {
		u.Admin = true
	} else {
		u.Admin = false
	}

	if isStaff == 1 {
		u.Staff = true
	} else {
		u.Staff = false
	}

	return
}
