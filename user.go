package main

import (
	"database/sql"
	"log"
)

// User -- struct for storing information regarding a user performing a query
type User struct {
	ID    int
	Name  string
	Admin bool
	Staff bool
}

func getUserFromToken(accessToken string) (User, error) {
	var user User
	var isAdmin int
	var isStaff int

	query := "SELECT auth_user.id, auth_user.username, auth_user.is_superuser, auth_user.is_staff FROM auth_user INNER JOIN authtoken_token ON auth_user.id = authtoken_token.user_id WHERE authtoken_token.key = ?"

	dq, err := dbConn.Prepare(query)

	if err != nil {
		log.Fatal(err)
	}

	defer dq.Close()

	err = dq.QueryRow(accessToken).Scan(&user.ID, &user.Name, &isAdmin, &isStaff)
	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, nil
		}
		log.Fatal(err)
	}

	if isAdmin == 1 {
		user.Admin = true
	} else {
		user.Admin = false
	}

	if isStaff == 1 {
		user.Staff = true
	} else {
		user.Staff = false
	}

	return user, nil
}
