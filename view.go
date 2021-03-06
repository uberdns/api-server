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
	fmt.Fprintf(w, "Welcome home")
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
	user := getUserFromRequest(r)

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

// To-do: Check for existing records
func createRecordView(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)

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

}

func listRecordView(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)

	if (User{}) == user {
		fmt.Println("User not found")
		// Empty user returned from token lookup - implied user not found
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("403 - Forbidden"))
		return
	}

	records := user.GetRecords(&dbConn)
	recordJSON, err := json.Marshal(records)
	if err != nil {
		log.Fatal(err)
	}
	w.Write([]byte(recordJSON))
}

func listAllRecordView(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)

	if (User{}) == user {
		fmt.Println("User not found")
		// Empty user returned from token lookup - implied user not found
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("403 - Forbidden"))
		return
	}

	// if requesting user is not an admin or staff, forbid access to ALL records
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

func deleteRecordView(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)

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

func purgeCacheView(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)

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

func purgeCacheRecordView(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)

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

// To-do: Check for existing records
func createDomainView(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)

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

func listDomainView(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)

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

func deleteDomainView(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)

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

func loginView(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Add("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		return
	case "GET":
		fmt.Println("Should redirect to index")
	case "POST":
		decoder := json.NewDecoder(r.Body)

		type Credentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		var credentials = Credentials{}
		if err := decoder.Decode(&credentials); err != nil {
			log.Fatal(err)
		}

		var user = User{}
		user.LookupFromName(credentials.Username)
		if user.IsPasswordAuthenticated(credentials.Password, &dbConn) {
			var jwtTokens = JWTTokens{}
			jwtTokens.New(user.ID)

			http.SetCookie(w, &http.Cookie{
				Name:       "token",
				Value:      jwtTokens.AccessToken.String(),
				Expires:    time.Now().Add(300 * time.Second),
				RawExpires: time.Now().Add(300 * time.Second).String(),
				MaxAge:     300,
			})
			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Write([]byte(jwtTokens.String()))
			return
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
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

func userProfileView(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case "GET":

		user := getUserFromRequest(r)

		if (User{}) == user {
			// Empty user returned from token lookup - implied user not found
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Forbidden"))
			return
		}
		type UserProfile struct {
			ID      int      `json:"id"`
			Name    string   `json:"name"`
			Records []Record `json:"records"`
		}
		userProfile := UserProfile{}
		userProfile.ID = user.ID
		userProfile.Name = user.Name
		userProfile.Records = user.GetRecords(&dbConn)
		recordsJSON, err := json.Marshal(userProfile)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(recordsJSON))
		return
	}
}
