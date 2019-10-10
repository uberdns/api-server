package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// Record -- struct for storing information regarding records
type Record struct {
	ID       int
	Name     string
	IP       string
	TTL      int64 //TTL for caching
	Created  time.Time
	DomainID int
	OwnerID  int
}

func recordExists(dbConn *sql.DB, record Record) bool {
	var ret int
	query := "SELECT COUNT(*) FROM dns_record WHERE name = ? and domain_id = ?"
	err := dbConn.QueryRow(query, record.Name, record.DomainID).Scan(&ret)

	if err != nil {
		log.Fatal(err)
	}

	if ret > 0 {
		return true
	}
	return false
}

func createRecord(dbConn *sql.DB, record Record) error {
	if !recordExists(dbConn, record) {
		query := "INSERT INTO dns_record (name, ip_address, ttl, created_on, domain_id, owner_id) VALUES (?, ?, ?,  ?, ?, ?)"
		dq, err := dbConn.Prepare(query)
		if err != nil {
			return err
		}

		defer dq.Close()

		_, err = dq.Exec(record.Name, record.IP, record.TTL, record.Created, record.DomainID, record.OwnerID)
		if err != nil {
			return err
		}
		return nil
	}
	return nil

}

func updateRecord(dbConn *sql.DB, record Record, newIPAddress string) error {
	query := "UPDATE dns_record SET ip_address = ? WHERE id = ?"
	dq, err := dbConn.Prepare(query)

	if err != nil {
		return err
	}

	defer dq.Close()

	_, err = dq.Exec(newIPAddress, record.ID)

	if err != nil {
		return err
	}

	return nil
}

func deleteRecord(dbConn *sql.DB, record Record) error {
	query := "DELETE FROM dns_record WHERE id = ?"
	dq, err := dbConn.Prepare(query)

	if err != nil {
		return err
	}

	defer dq.Close()

	_, err = dq.Exec(record.ID)

	if err != nil {
		return err
	}

	return nil
}

func getRecordFromFQDN(fqdn string) (Record, error) {
	var record Record

	recordName := strings.Split(fqdn, ".")[0]
	topLevelDomain := strings.Join(strings.Split(fqdn, ".")[1:], ".")

	domain, err := getDomainFromName(topLevelDomain)

	if err != nil {
		log.Fatal(err)
	}

	query := "SELECT id, name, ip_address, ttl, created_on, owner_id FROM dns_record WHERE name = ? AND domain_id = ?"

	dq, err := dbConn.Prepare(query)

	if err != nil {
		log.Fatal(err)
	}

	defer dq.Close()

	err = dq.QueryRow(recordName, domain.ID).Scan(&record.ID, &record.Name, &record.IP, &record.TTL, &record.Created, &record.OwnerID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Unable to find record with that FQDN.")
			return Record{}, nil
		}
		log.Fatal(err)
	}

	return record, nil
}
