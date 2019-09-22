//To-Do:
// - Prometheus stats
// - debug logging
// - on record create, check for record existence

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"gopkg.in/ini.v1"
)

// Record -- struct for storing information regarding records
type Record struct {
	ID       int
	Name     string
	IP       string
	TTL      int64 //TTL for caching
	DOB      time.Time
	DomainID int
	OwnerID  int
}

type Domain struct {
	ID        int
	Name      string
	CreatedOn time.Time
}

type User struct {
	ID    int
	Name  string
	Admin bool
	Staff bool
}

type requestRecord struct {
	FQDN      string
	IPAddress string
}

type FQDN struct {
	FQDN string
}

var dbConn sql.DB
var redisClient *redis.Client
var redisCacheChannelName string

func dbConnect(username string, password string, host string, port int, database string) error {
	conn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", username, password, host, port, database)
	dbc, err := sql.Open("mysql", conn)

	if err != nil {
		return err
	}

	//defer dbc.Close()

	err = dbc.Ping()
	if err != nil {
		panic(err.Error())
	}

	dbConn = *dbc
	return nil
}

func isValidRequest(w http.ResponseWriter, r *http.Request) bool {
	valid := false

	// Iterate over keylist of headers and make sure
	// our required headers are present.
	for k, _ := range r.Header {
		if k == "X-Api-Key" {
			valid = true
		}
	}

	if !valid {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("403 - Forbidden"))
	}

	return valid
}

func createRecord(dbConn *sql.DB, record Record) error {
	query := "INSERT INTO dns_record (name, ip_address, ttl, created_on, domain_id, owner_id) VALUES (?, ?, ?,  ?, ?, ?)"
	dq, err := dbConn.Prepare(query)
	if err != nil {
		return err
	}

	defer dq.Close()

	_, err = dq.Exec(record.Name, record.IP, record.TTL, record.DOB, record.DomainID, record.OwnerID)
	if err != nil {
		return err
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

func createDomain(dbConn *sql.DB, domain Domain) error {
	query := "INSERT INTO dns_domain (name, created_on) VALUES (?, ?)"
	dq, err := dbConn.Prepare(query)
	if err != nil {
		return err
	}

	defer dq.Close()

	_, err = dq.Exec(domain.Name, time.Now())
	if err != nil {
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

func getUserFromToken(accessToken string) (User, error) {
	var user User
	var isAdmin int
	var isStaff int

	query := "SELECT auth_user.id, auth_user.username, auth_user.is_superuser, auth_user.is_staff FROM auth_user INNER JOIN authtoken_token ON auth_user.id = authtoken_token.user_id WHERE authtoken_token.key = ?"

	dq, err := dbConn.Prepare(query)

	if err != nil {
		log.Fatal(err)
	}

	defer dq.Close()

	err = dq.QueryRow(accessToken).Scan(&user.ID, &user.Name, &isAdmin, &isStaff)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Unable to find user with provided API token.")
			return User{}, err
		} else {
			log.Fatal(err)
		}
	}

	if isAdmin == 1 {
		user.Admin = true
	} else {
		user.Admin = false
	}

	if isStaff == 1 {
		user.Staff = true
	} else {
		user.Staff = false
	}

	return user, nil
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
		} else {
			log.Fatal(err)
		}
	}

	return domain, nil
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
		} else {
			log.Fatal(err)
		}
	}

	return domain, nil
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

	err = dq.QueryRow(recordName, domain.ID).Scan(&record.ID, &record.Name, &record.IP, &record.TTL, &record.DOB, &record.OwnerID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Unable to find record with that FQDN.")
			return Record{}, nil
		} else {
			log.Fatal(err)
		}
	}

	return record, nil
}

// This is used to determine if the request is authorized
// to update the requested record
func isAllowed(accessToken string, record Record) bool {
	user, err := getUserFromToken(accessToken)
	if err != nil {
		log.Fatal(err)
	}

	if user.ID == record.OwnerID {
		return true
	}

	return false
}

func purgeCache(cacheChannel string, record Record) error {

	recordJSON, err := json.Marshal(record)
	if err != nil {
		return err
	}

	err = redisClient.Publish(cacheChannel, recordJSON).Err()
	if err != nil {
		return err
	}
	return nil
}

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

			record, err := getRecordFromFQDN(reqRecord.FQDN)
			if err != nil {
				log.Fatal(err)
			}

			if isAllowed(apiKey, record) {
				err := updateRecord(&dbConn, record, reqRecord.IPAddress)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Fprintf(w, "Record was updated successfully")

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

			recordName := strings.Split(reqRecord.FQDN, ".")[0]
			domainName := strings.Join(strings.Split(reqRecord.FQDN, ".")[1:], ".")

			domain, err := getDomainFromName(domainName)
			if err != nil {
				log.Fatal(err)
			}

			record := Record{
				Name:     recordName,
				IP:       reqRecord.IPAddress,
				TTL:      30,
				DOB:      time.Now(),
				DomainID: domain.ID,
				OwnerID:  owner.ID,
			}

			err = createRecord(&dbConn, record)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(w, "Record was created successfully: %s", reqRecord.FQDN)
		}
	}
}

func deleteRecordView(w http.ResponseWriter, r *http.Request) {
	if isValidRequest(w, r) {
		accessToken := fmt.Sprintf("%s", r.Header["X-Api-Key"])
		accessToken = strings.Trim(accessToken, "[")
		accessToken = strings.Trim(accessToken, "]")

		var reqRecord FQDN

		switch r.Method {
		case "GET":
			fmt.Println("should redirect to index on GET request")
		case "DELETE":
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
				err = deleteRecord(&dbConn, record)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Fprintf(w, "Record was deleted successfully")
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

		// Only admin + staff can purge the entire cache
		if !user.Admin && !user.Staff {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 - Forbidden"))
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
				err := purgeCache(redisCacheChannelName, record)
				if err != nil {
					log.Fatal(err)
				}
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

		// if requesting user is not an admin or staff, forbid access
		if !user.Admin {
			if !user.Staff {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("403 - Forbidden"))
			}
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

		// if requesting user is not an admin or staff, forbid access
		if !user.Admin {
			if !user.Staff {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("403 - Forbidden"))
			}
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

func main() {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		panic(err.Error())
	}

	dbHost := cfg.Section("database").Key("host").String()
	dbUser := cfg.Section("database").Key("user").String()
	dbPass := cfg.Section("database").Key("pass").String()
	dbPort, _ := cfg.Section("database").Key("port").Int()
	dbName := cfg.Section("database").Key("database").String()

	redisHost := cfg.Section("redis").Key("host").String()
	redisPassword := cfg.Section("redis").Key("password").String()
	redisDB, _ := cfg.Section("redis").Key("db").Int()
	redisCacheChannelName = cfg.Section("redis").Key("cache_channel").String()

	err = dbConnect(dbUser, dbPass, dbHost, dbPort, dbName)
	if err != nil {
		panic(err.Error())
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Password: redisPassword,
		DB:       redisDB,
	})

	go func() {
		for {
			_, err := redisClient.Ping().Result()
			if err != nil {
				fmt.Println("Redis is broken")
			}
			time.Sleep(time.Second)
		}
	}()

	go func() {
		redisCacheChannel := redisClient.Subscribe(redisCacheChannelName)
		_, err := redisCacheChannel.Receive()
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		r := http.NewServeMux()
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)
		r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))

		http.ListenAndServe(":6060", r)
	}()

	go func() {
		router := mux.NewRouter().StrictSlash(true)
		router.HandleFunc("/", indexView)
		router.HandleFunc("/cache/purge", purgeCacheView)
		router.HandleFunc("/cache/record/purge", purgeCacheRecordView)
		router.HandleFunc("/domain/create", createDomainView)
		router.HandleFunc("/domain/delete", deleteDomainView)
		router.HandleFunc("/record/create", createRecordView)
		router.HandleFunc("/record/update", updateRecordView)
		router.HandleFunc("/record/delete", deleteRecordView)
		log.Fatal(http.ListenAndServe(":8080", router))
	}()

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Fatalf("Signal (%v) received, stopping\n", s)

}
