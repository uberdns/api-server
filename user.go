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

func (u *User) IsPasswordAuthenticated(password string, dbConn *sql.DB) bool {
	var realPass string

	query := "SELECT password FROM auth_user WHERE id = ?"

	dq, err := dbConn.Prepare(query)

	if err != nil {
		log.Fatal(err)
	}

	defer dq.Close()

	if err = dq.QueryRow(u.ID).Scan(&realPass); err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		log.Fatal(err)
	}

	valid, err := NewPBKDF2SHA256Hasher().Verify(password, realPass)

	if err != nil {
		log.Fatal(err)
	}

	return valid

}

func (u *User) GetRecords(dbConn *sql.DB) []Record {
	var records []Record
	query := "SELECT id, name, ip_address, ttl, created_on FROM dns_record WHERE owner_id = ?"

	dq, err := dbConn.Prepare(query)

	rows, err := dq.Query(u.ID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		record := Record{}
		if err := rows.Scan(&record.ID, &record.Name, &record.IP, &record.TTL, &record.CreatedOn); err != nil {
			log.Fatal(err)
		}
		// if record lookup successful, tag record as ours for return
		record.OwnerID = u.ID
		records = append(records, record)
	}

	return records
}

func (u *User) LookupFromName(username string) {
	var isAdmin int
	var isStaff int

	query := "SELECT id, is_superuser, is_staff FROM auth_user WHERE username = ?"

	dq, err := dbConn.Prepare(query)

	if err != nil {
		log.Fatal(err)
	}

	defer dq.Close()

	if err = dq.QueryRow(username).Scan(&u.ID, &isAdmin, &isStaff); err != nil {
		if err == sql.ErrNoRows {
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

func (u *User) HasAPIKey(apiKey string, dbConn *sql.DB) bool {
	query := "SELECT COUNT(*) FROM auth_user INNER JOIN authtoken_token ON auth_user.id = authtoken_token.user_id WHERE authtoken_token.key = ?"

	var count int
	dq, err := dbConn.Prepare(query)

	if err != nil {
		log.Fatal(err)
	}
	defer dq.Close()

	if err := dq.QueryRow(apiKey).Scan(&count); err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		log.Fatal(err)
	}

	if count == 1 {
		return true
	}
	log.Fatalf("Error: Duplicate API keys detected in database - %s", apiKey)
	return true
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

func (u *User) LookupFromID() {
	var isAdmin int
	var isStaff int

	query := "SELECT username, is_superuser, is_staff FROM auth_user WHERE id = ?"

	dq, err := dbConn.Prepare(query)

	if err != nil {
		log.Fatal(err)
	}

	defer dq.Close()

	err = dq.QueryRow(u.ID).Scan(&u.Name, &isAdmin, &isStaff)
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
