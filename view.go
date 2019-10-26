package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func indexView(w http.ResponseWriter, r *http.Request) {
	fmt.Println("validating request")
	if isValidRequest(w, r) {
		fmt.Fprintf(w, "Welcome home")
	}
}

func createJWTTokenView(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		fmt.Println(" should redirect to index")
	case "POST":
		type JWTRequest struct {
			Username string
			Password string
		}
		var jwtRequest = JWTRequest{}
		decoder := json.NewDecoder(r.Body)

		if err := decoder.Decode(&jwtRequest); err != nil {
			log.Fatal(err)
		}

		var user = User{}
		user.LookupFromName(jwtRequest.Username)

		if (User{}) == user {
			fmt.Printf("Username %s not found", jwtRequest.Username)
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Forbidden"))
			return
		}

		if user.IsPasswordAuthenticated(jwtRequest.Password, &dbConn) {
			jwtToken := JWTTokens{}
			jwtToken.New(user.ID)
			fmt.Println(jwtToken.String())
		}

	}
}

func updateRecordView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := getAPIKey(r)
		user := User{}
		user.LookupFromAPIKey(accessToken)

		var reqRecord requestRecord
		switch r.Method {
		case "GET":
			fmt.Println("should redirect to index on GET request")
		case "POST":
			decoder := json.NewDecoder(r.Body)

			err := decoder.Decode(&reqRecord)

			if err != nil {
				log.Fatal(err)
			}

			record := Record{}

			if err = record.LookupFromFQDN(reqRecord.Name); err != nil {
				log.Fatal(err)
			}
			if record.IsUserAllowed(user) {
				record.IP = reqRecord.IPAddress
				if err = record.Save(&dbConn); err != nil {
					log.Fatal(err)
				}
				fmt.Fprintf(w, "Record was updated successfully")
				if err := record.Purge(recordChannel); err != nil {
					log.Print("Unable to send purge record message to redis")
					log.Fatal(err)
				}

				if err := record.Cache(recordChannel); err != nil {
					log.Fatal(err)
				}
			} else {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("403 - Forbidden"))
			}
		}
	}
}

// To-do: Check for existing records
func createRecordView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := getAPIKey(r)
		user := User{}
		user.LookupFromAPIKey(accessToken)

		var reqRecord requestRecord

		switch r.Method {
		case "GET":
			fmt.Println("should redirect to index on GET request")
		case "POST":
			decoder := json.NewDecoder(r.Body)

			err := decoder.Decode(&reqRecord)

			if err != nil {
				log.Fatal(err)
			}

			if (User{}) == user {
				// Empty user returned from token lookup - implied user not found
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("403 - Forbidden"))
				return
			}

			recordName := strings.Split(reqRecord.Name, ".")[0]
			domainName := strings.Join(strings.Split(reqRecord.Name, ".")[1:], ".")

			domain := Domain{}
			if err := domain.LookupFromFQDN(domainName); err != nil {
				log.Fatal(err)
			}

			record := Record{
				Name:      recordName,
				IP:        reqRecord.IPAddress,
				TTL:       30,
				CreatedOn: time.Now(),
				DomainID:  domain.ID,
				OwnerID:   user.ID,
			}

			if err = record.Save(&dbConn); err != nil {
				log.Fatal(err)
			}
			// Perform SQL Query after creating the record to pull autoincrement values
			// this is kinda clunky...but its on the api server so im not super worried...
			// this might be worthy of refactoring at some point for brownie points
			if err = record.LookupFromFQDN(reqRecord.Name); err != nil {
				fmt.Println("problem querying database after creating record successfully")
				log.Fatal(err)
			}
			fmt.Fprintf(w, "Record was created successfully: %s", reqRecord.Name)
			if err = record.Cache(recordChannel); err != nil {
				log.Fatal(err)
			}

			fmt.Println(record)
		}
	} else {
		fmt.Println("invalid request")
	}
}

func listRecordView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := getAPIKey(r)
		user := User{}
		user.LookupFromAPIKey(accessToken)

		if (User{}) == user {
			// Empty user returned from token lookup - implied user not found
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Forbidden"))
			return
		}

		// if requesting user is not an admin or staff, forbid access
		if !user.Admin && !user.Staff {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Forbidden"))
			return
		}

		records := listRecords(&dbConn)
		recordJSON, err := json.Marshal(records)
		if err != nil {
			log.Fatal(err)
		}
		w.Write([]byte(recordJSON))
	}
}

func deleteRecordView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := getAPIKey(r)
		user := User{}
		user.LookupFromAPIKey(accessToken)

		var reqRecord requestRecord

		switch r.Method {
		case "GET":
			fmt.Println("should redirect to index on GET request")
		case "DELETE":
			decoder := json.NewDecoder(r.Body)

			err := decoder.Decode(&reqRecord)

			if err != nil {
				log.Fatal(err)
			}

			record := Record{}

			if err = record.LookupFromFQDN(reqRecord.Name); err != nil {
				log.Fatal(err)
			}

			if record.IsUserAllowed(user) {
				if err = record.Delete(&dbConn); err != nil {
					log.Fatal(err)
				}
				if err = record.Purge(recordChannel); err != nil {
					log.Fatal(err)
				}
				w.Write([]byte("Record purged from cache"))
				fmt.Println("Record purged from cache")
			}
		}

	}
}

func purgeCacheView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := getAPIKey(r)
		user := User{}
		user.LookupFromAPIKey(accessToken)

		if (User{}) == user {
			// Empty user returned from token lookup - implied user not found
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Forbidden"))
			return
		}

		// Only admin + staff can purge the entire cache
		if !user.Admin && !user.Staff {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Forbidden"))
			return
		}

		type FQDN struct {
			FQDN string
		}

		var reqRecord FQDN

		switch r.Method {
		case "GET":
			fmt.Println("should redirect to index on GET request")
		case "POST":
			decoder := json.NewDecoder(r.Body)

			err := decoder.Decode(&reqRecord)

			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("To-Do: create redis message to clean all known records from cache")

			fmt.Fprintf(w, "Record cached from purge globally. Please allow up to 30 seconds for this to reflect.")
		}
	}
}

func purgeCacheRecordView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := getAPIKey(r)
		user := User{}
		user.LookupFromAPIKey(accessToken)

		type FQDN struct {
			FQDN string
		}
		var reqRecord FQDN

		switch r.Method {
		case "GET":
			fmt.Println("should redirect to index on GET request")
		case "POST":
			decoder := json.NewDecoder(r.Body)

			err := decoder.Decode(&reqRecord)

			if err != nil {
				log.Fatal(err)
			}

			record := Record{}

			if err = record.LookupFromFQDN(reqRecord.FQDN); err != nil {
				log.Fatal(err)
			}

			if record.IsUserAllowed(user) {
				if err = record.Purge(recordChannel); err != nil {
					log.Fatal(err)
				}
				w.Write([]byte("Record purged from cache"))
			}

			fmt.Fprintf(w, "Record cached from purge globally. Please allow up to 30 seconds for this to reflect.")
		}
	}
}

// To-do: Check for existing records
func createDomainView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := getAPIKey(r)
		user := User{}
		user.LookupFromAPIKey(accessToken)

		if (User{}) == user {
			// Empty user returned from token lookup - implied user not found
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Forbidden"))
			return
		}

		// if requesting user is not an admin or staff, forbid access
		if !user.Admin && !user.Staff {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Forbidden"))
			return
		}

		type reqDomain struct {
			Name string
		}

		var requestDomain reqDomain

		switch r.Method {
		case "GET":
			fmt.Println("should redirect to index on GET request")
		case "POST":
			decoder := json.NewDecoder(r.Body)

			err := decoder.Decode(&requestDomain)

			if err != nil {
				log.Fatal(err)
			}

			domain := Domain{
				Name: requestDomain.Name,
			}

			if err = domain.Save(&dbConn); err != nil {
				log.Fatal(err)
			}

			fmt.Fprintf(w, "Domain was created successfully: %s", domain.Name)

			if err = domain.Cache(domainChannel); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func listDomainView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := getAPIKey(r)
		user := User{}
		user.LookupFromAPIKey(accessToken)

		if (User{}) == user {
			// Empty user returned from token lookup - implied user not found
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Forbidden"))
			return
		}

		// if requesting user is not an admin or staff, forbid access
		if !user.Admin && !user.Staff {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Forbidden"))
			return
		}

		domains := listDomains(&dbConn)
		domainJSON, err := json.Marshal(domains)
		if err != nil {
			log.Fatal(err)
		}
		w.Write([]byte(domainJSON))

	}
}

func deleteDomainView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := getAPIKey(r)
		user := User{}
		user.LookupFromAPIKey(accessToken)

		if (User{}) == user {
			// Empty user returned from token lookup - implied user not found
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Forbidden"))
			return
		}

		// if requesting user is not an admin or staff, forbid access
		if !user.Admin && !user.Staff {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Forbidden"))
			return
		}

		type requestDomain struct {
			Name string
		}

		var reqDomain requestDomain

		switch r.Method {
		case "GET":
			fmt.Println("should redirect to index on GET request")
		case "DELETE":
			decoder := json.NewDecoder(r.Body)

			err := decoder.Decode(&reqDomain)

			if err != nil {
				log.Fatal(err)
			}

			domain := Domain{}
			if err := domain.LookupFromFQDN(reqDomain.Name); err != nil {
				log.Fatal(err)
			}

			if err = domain.Delete(&dbConn); err != nil {
				log.Fatal(err)
			}

			fmt.Fprintf(w, "Domain was deleted successfully")
		}
	}
}

func loginView(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Header)

	switch r.Method {
	case "GET":
		fmt.Println("Should redirect to index")
	case "POST":
		fmt.Println("nice")
	}
	user := User{ID: 1}
	token, err := getTokenForUser(&dbConn, user)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(w, "{\"token\": \"%s\"}", token)
	fmt.Println(token)

}

func logoutView(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		fmt.Println("Should redirect to index")
	case "POST":
		fmt.Println("nice")
	}
	fmt.Println(r.Header)
	token := Token{
		Key: "123",
	}
	err := tokenInvalidate(&dbConn, token)
	if err != nil {
		log.Fatal(err)
	}
}
