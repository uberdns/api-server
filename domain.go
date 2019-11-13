package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

func (d *Domain) Save(dbConn *sql.DB) error {
	query := "INSERT INTO dns_domain (name, created_on) VALUES (?, ?)"
	dq, err := dbConn.Prepare(query)
	if err != nil {
		return err
	}

	defer dq.Close()

	_, err = dq.Exec(d.Name, time.Now())
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func (d *Domain) Delete(dbConn *sql.DB) error {
	query := "DELETE FROM dns_domain WHERE id = ?"
	dq, err := dbConn.Prepare(query)
	if err != nil {
		return err
	}

	defer dq.Close()

	_, err = dq.Exec(d.ID)
	if err != nil {
		return err
	}

	return nil
}

func (d *Domain) Cache(channel chan<- CacheControlMessage) error {
	jsonMSG, err := json.Marshal(d)
	if err != nil {
		return err
	}

	msg := CacheControlMessage{
		Action: "create",
		Type:   "domain",
		Object: string(jsonMSG),
	}

	channel <- msg
	return nil
}

func (d *Domain) Purge(channel chan<- CacheControlMessage) error {
	jsonMSG, err := json.Marshal(d)
	if err != nil {
		return err
	}

	msg := CacheControlMessage{
		Action: "purge",
		Type:   "domain",
		Object: string(jsonMSG),
	}

	channel <- msg
	return nil
}

func listDomains(dbConn *sql.DB) []Domain {
	var domains []Domain
	query := "SELECT id, name, created_on FROM dns_domain"

	rows, err := dbConn.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		domain := Domain{}
		if err := rows.Scan(&domain.ID, &domain.Name, &domain.CreatedOn); err != nil {
			log.Fatal(err)
		}
		domains = append(domains, domain)
	}

	return domains
}

func (d *Domain) LookupFromID(id int) error {
	query := "SELECT id, name, created_on FROM dns_domain WHERE id = ?"

	dq, err := dbConn.Prepare(query)

	if err != nil {
		return err
	}

	defer dq.Close()
	err = dq.QueryRow(id).Scan(&d.ID, &d.Name, &d.CreatedOn)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Unable to find domain with provided domain id: ", id)
			return nil
		}
		return err
	}

	return nil
}

func (d *Domain) LookupFromFQDN(fqdn string) error {
	query := "SELECT id, name, created_on FROM dns_domain WHERE name = ?"

	dq, err := dbConn.Prepare(query)

	if err != nil {
		return err
	}

	defer dq.Close()
	err = dq.QueryRow(fqdn).Scan(&d.ID, &d.Name, &d.CreatedOn)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Unable to find domain with provided domain name: ", fqdn)
			return nil
		}
		return err
	}

	return nil
}
