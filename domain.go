package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Domain -- struct for storing information regarding domains
type Domain struct {
	ID        int
	Name      string
	CreatedOn time.Time
}

func createDomain(dbConn *sql.DB, domain Domain) error {
	query := "INSERT INTO dns_domain (name, created_on) VALUES (?, ?)"
	dq, err := dbConn.Prepare(query)
	if err != nil {
		return err
	}

	defer dq.Close()

	_, err = dq.Exec(domain.Name, time.Now())
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func deleteDomain(dbConn *sql.DB, domain Domain) error {
	query := "DELETE FROM dns_domain WHERE id = ?"
	dq, err := dbConn.Prepare(query)
	if err != nil {
		return err
	}

	defer dq.Close()

	_, err = dq.Exec(domain.ID)
	if err != nil {
		return err
	}

	return nil
}

func getDomainFromID(domainID int) (Domain, error) {
	var domain Domain

	query := "SELECT id, name, created_on FROM dns_domain WHERE id = ?"

	dq, err := dbConn.Prepare(query)

	if err != nil {
		log.Fatal(err)
	}

	defer dq.Close()
	err = dq.QueryRow(domainID).Scan(&domain.ID, &domain.Name, &domain.CreatedOn)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Unable to find domain with provided domain id: ", domainID)
			return Domain{}, nil
		}
		log.Fatal(err)
	}

	return domain, nil
}

func getDomainFromName(domainName string) (Domain, error) {
	var domain Domain

	query := "SELECT id, name, created_on FROM dns_domain WHERE name = ?"

	dq, err := dbConn.Prepare(query)

	if err != nil {
		log.Fatal(err)
	}

	defer dq.Close()
	err = dq.QueryRow(domainName).Scan(&domain.ID, &domain.Name, &domain.CreatedOn)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Unable to find domain with provided domain name: ", domainName)
			return Domain{}, nil
		}
		log.Fatal(err)
	}

	return domain, nil
}
