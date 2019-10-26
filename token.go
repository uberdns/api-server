package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type Token struct {
	Key       string
	CreatedOn time.Time
	UserID    int64
}

// Re-write this as one large exists and valid check

func tokenExists(dbConn *sql.DB, user User) bool {
	var ret int
	query := "SELECT COUNT(*) FROM authtoken_token WHERE user_id = ?"
	err := dbConn.QueryRow(query, user.ID).Scan(&ret)

	if err != nil {
		log.Fatal(err)
	}

	if ret > 0 {
		return true
	}
	return false
}

func tokenIsValid(dbConn *sql.DB, token Token) bool {
	var retToken Token
	query := "SELECT key, created, user_id FROM authtoken_token WHERE key = ?"
	err := dbConn.QueryRow(query, token.Key).Scan(&retToken.Key, &retToken.CreatedOn, &retToken.UserID)
	if err != nil {
		log.Fatal(err)
	}

	hourPast := time.Now().AddDate(0, -1, 0)

	if retToken.CreatedOn.Before(hourPast) {
		fmt.Printf("Token %s expired", token.Key)
		return false
	}
	return true

}

func tokenInvalidate(dbConn *sql.DB, token Token) error {
	query := "DELETE FROM authtoken_token WHERE key = ?"
	dq, err := dbConn.Prepare(query)

	defer dq.Close()

	_, err = dq.Exec(token.Key)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func getTokenForUser(dbConn *sql.DB, user User) (string, error) {
	var token string
	if tokenExists(dbConn, user) {
		query := "SELECT `key` FROM authtoken_token WHERE user_id = ?"
		err := dbConn.QueryRow(query, user.ID).Scan(&token)
		if err != nil {
			log.Fatal(err)
		}

		return token, nil
	}
	token = "randomstringofcharacters123123123"
	query := "INSERT INTO authtoken_token (`key`, created, user_id) VALUES (?, ?, ?)"
	dq, err := dbConn.Prepare(query)

	defer dq.Close()

	_, err = dq.Exec(token, time.Now(), user.ID)
	if err != nil {
		log.Fatal(err)
	}
	return token, nil
}
