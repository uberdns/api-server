package main

import (
	"database/sql"
	"fmt"
)

var dbConn sql.DB

func dbConnect(username string, password string, host string, port int, database string) error {
	conn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", username, password, host, port, database)
	dbc, err := sql.Open("mysql", conn)

	if err != nil {
		return err
	}

	err = dbc.Ping()
	if err != nil {
		panic(err.Error())
	}

	dbConn = *dbc
	return nil
}
