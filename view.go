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

func updateRecordView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		apiKey := fmt.Sprintf("%s", r.Header["X-Api-Key"])
		apiKey = strings.Trim(apiKey, "[")
		apiKey = strings.Trim(apiKey, "]")

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

			record, err := getRecordFromFQDN(reqRecord.Name)
			if err != nil {
				log.Fatal(err)
			}

			if isAllowed(apiKey, record) {
				err := updateRecord(&dbConn, record, reqRecord.IPAddress)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Fprintf(w, "Record was updated successfully")
				err = recordCacheMsgHandler(redisCacheChannelName, "update", record)
				if err != nil {
					log.Printf(err.Error())
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
		accessToken := fmt.Sprintf("%s", r.Header["X-Api-Key"])
		accessToken = strings.Trim(accessToken, "[")
		accessToken = strings.Trim(accessToken, "]")

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

			owner, err := getUserFromToken(accessToken)
			if err != nil {
				log.Fatal(err)
			}
			if (User{}) == owner {
				// Empty user returned from token lookup - implied user not found
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("403 - Forbidden"))
				return
			}

			recordName := strings.Split(reqRecord.Name, ".")[0]
			domainName := strings.Join(strings.Split(reqRecord.Name, ".")[1:], ".")

			domain, err := getDomainFromName(domainName)
			if err != nil {
				log.Fatal(err)
			}

			record := Record{
				Name:     recordName,
				IP:       reqRecord.IPAddress,
				TTL:      30,
				Created:  time.Now(),
				DomainID: domain.ID,
				OwnerID:  owner.ID,
			}

			err = createRecord(&dbConn, record)
			if err != nil {
				log.Fatal(err)
			}
			// Perform SQL Query after creating the record to pull autoincrement values
			// this is kinda clunky...but its on the api server so im not super worried...
			// this might be worthy of refactoring at some point for brownie points
			record, err = getRecordFromFQDN(reqRecord.Name)
			if err != nil {
				fmt.Println("problem querying database after creating record successfully")
				log.Fatal(err)
			}
			fmt.Fprintf(w, "Record was created successfully: %s", reqRecord.Name)
			addRecordToCache(record, recordChannel)
			//err = recordCacheMsgHandler(redisCacheChannelName, "create", record)
			//if err != nil {
			//	fmt.Printf("Unable to populate cache with record %s", record.Name)
			//}
			fmt.Println(record)
		}
	} else {
		fmt.Println("invalid request")
	}
}

func deleteRecordView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := fmt.Sprintf("%s", r.Header["X-Api-Key"])
		accessToken = strings.Trim(accessToken, "[")
		accessToken = strings.Trim(accessToken, "]")

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

			record, err := getRecordFromFQDN(reqRecord.Name)

			if err != nil {
				log.Fatal(err)
			}

			if isAllowed(accessToken, record) {
				err = deleteRecord(&dbConn, record)
				if err != nil {
					log.Fatal(err)
				}
				deleteRecordFromCache(record, recordChannel)
			}
		}

	}
}

func purgeCacheView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := fmt.Sprintf("%s", r.Header["X-Api-Key"])
		accessToken = strings.Trim(accessToken, "[")
		accessToken = strings.Trim(accessToken, "]")

		user, err := getUserFromToken(accessToken)
		if err != nil {
			log.Fatal(err)
		}
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
		accessToken := fmt.Sprintf("%s", r.Header["X-Api-Key"])
		accessToken = strings.Trim(accessToken, "[")
		accessToken = strings.Trim(accessToken, "]")

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

			record, err := getRecordFromFQDN(reqRecord.FQDN)
			if err != nil {
				log.Fatal(err)
			}

			if isAllowed(accessToken, record) {
				deleteRecordFromCache(record, recordChannel)
			}

			fmt.Fprintf(w, "Record cached from purge globally. Please allow up to 30 seconds for this to reflect.")
		}
	}
}

// To-do: Check for existing records
func createDomainView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := fmt.Sprintf("%s", r.Header["X-Api-Key"])
		accessToken = strings.Trim(accessToken, "[")
		accessToken = strings.Trim(accessToken, "]")

		user, err := getUserFromToken(accessToken)
		if err != nil {
			log.Fatal(err)
		}
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

			err = createDomain(&dbConn, domain)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Fprintf(w, "Domain was created successfully: %s", domain.Name)
		}
	}
}

func deleteDomainView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := fmt.Sprintf("%s", r.Header["X-Api-Key"])
		accessToken = strings.Trim(accessToken, "[")
		accessToken = strings.Trim(accessToken, "]")

		user, err := getUserFromToken(accessToken)
		if err != nil {
			log.Fatal(err)
		}
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

			domain, err := getDomainFromName(reqDomain.Name)
			if err != nil {
				log.Fatal(err)
			}

			err = deleteDomain(&dbConn, domain)
			if err != nil {
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
