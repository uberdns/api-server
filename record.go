package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

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

func (r *Record) Save(dbConn *sql.DB) error {
	query := "INSERT INTO dns_record (name, ip_address, ttl, created_on, domain_id, owner_id) VALUES (?, ?, ?,  ?, ?, ?)"
	dq, err := dbConn.Prepare(query)
	if err != nil {
		return err
	}

	defer dq.Close()

	_, err = dq.Exec(r.Name, r.IP, r.TTL, r.CreatedOn, r.DomainID, r.OwnerID)
	if err != nil {
		return err
	}
	return nil
}

func (r *Record) Cache(channel chan<- CacheControlMessage) error {
	jsonMSG, err := json.Marshal(r)
	if err != nil {
		return err
	}

	msg := CacheControlMessage{
		Action: "create",
		Type:   "record",
		Object: string(jsonMSG),
	}
	channel <- msg
	return nil
}

func (r *Record) Delete(dbConn *sql.DB) error {
	query := "DELETE FROM dns_record WHERE id = ?"
	dq, err := dbConn.Prepare(query)

	if err != nil {
		return err
	}

	defer dq.Close()

	_, err = dq.Exec(r.ID)

	if err != nil {
		return err
	}

	return nil
}

func (r *Record) LookupFromFQDN(fqdn string) error {
	recordName := strings.Split(fqdn, ".")[0]
	topLevelDomain := strings.Join(strings.Split(fqdn, ".")[1:], ".")

	domain := Domain{}
	if err := domain.LookupFromFQDN(topLevelDomain); err != nil {
		log.Fatal(err)
	}

	query := "SELECT id, name, ip_address, ttl, created_on, owner_id FROM dns_record WHERE name = ? AND domain_id = ?"

	dq, err := dbConn.Prepare(query)

	if err != nil {
		log.Fatal(err)
	}

	defer dq.Close()

	err = dq.QueryRow(recordName, domain.ID).Scan(&r.ID, &r.Name, &r.IP, &r.TTL, &r.CreatedOn, &r.OwnerID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Unable to find record with that FQDN.")
			return nil
		}
		log.Fatal(err)
	}

	return nil
}

func (r *Record) Purge(channel chan<- CacheControlMessage) error {
	jsonMSG, err := json.Marshal(r)
	if err != nil {
		return err
	}

	msg := CacheControlMessage{
		Action: "purge",
		Type:   "record",
		Object: string(jsonMSG),
	}

	channel <- msg
	return nil
}

func (r *Record) IsUserAllowed(user User) bool {
	if r.OwnerID == user.ID {
		return true
	}

	return false
}

func listRecords(dbConn *sql.DB) []Record {
	var records []Record
	query := "SELECT id, name, ip_address, ttl, created_on, owner_id FROM dns_record"

	rows, err := dbConn.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		record := Record{}
		if err := rows.Scan(&record.ID, &record.Name, &record.IP, &record.TTL, &record.CreatedOn, &record.OwnerID); err != nil {
			log.Fatal(err)
		}
		records = append(records, record)
	}

	return records
}
