//To-Do:
// - Prometheus stats
// - debug logging
// - on record create, check for record existence

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"

	"gopkg.in/ini.v1"
)

// Domain -- struct for storing information regarding domains
type Domain struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedOn time.Time `json:"created_on"`
}

// Record -- struct for storing information regarding records
type Record struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	IP        string    `json:"ip"`
	TTL       int64     `json:"ttl"` //TTL for caching
	CreatedOn time.Time `json:"created_on"`
	DomainID  int       `json:"domain_id"`
	OwnerID   int       `json:"owner_id"`
}

type RequestCounter struct {
	Total int
	mu    sync.RWMutex
}

// CacheControlMessage -- struct for storing/parsing redis cache control messages
//  					  to the dns server
type CacheControlMessage struct {
	Action string
	Type   string
	Object string
}

var (
	domainChannel              chan CacheControlMessage
	recordChannel              chan CacheControlMessage
	redisCacheChannel          string
	requestCounter             RequestCounter
	unauthorizedRequestCounter RequestCounter
	prometheusPort             int
	encryptionSalt             string
)

func main() {
	cfgFile := flag.String("config", "config.ini", "Path to the config file")
	flag.Parse()

	iniOptions := ini.LoadOptions{
		IgnoreInlineComment: true, // ini craps out when a string contains a # char
	}

	cfg, err := ini.LoadSources(iniOptions, *cfgFile)
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
	redisCacheChannel = cfg.Section("redis").Key("cache_channel").String()

	apiPort, _ := cfg.Section("api").Key("api_port").Int()
	prometheusPort, _ = cfg.Section("api").Key("prometheus_port").Int()
	pprofPort, _ := cfg.Section("api").Key("pprof_port").Int()

	encryptionSalt = cfg.Section("security").Key("secret_key").String()

	go func() {
		r := http.NewServeMux()
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)
		r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))

		http.ListenAndServe(fmt.Sprintf(":%d", pprofPort), r)
	}()

	requestCounter = RequestCounter{
		Total: 0,
	}

	unauthorizedRequestCounter = RequestCounter{
		Total: 0,
	}

	err = dbConnect(dbUser, dbPass, dbHost, dbPort, dbName)
	if err != nil {
		panic(err.Error())
	}

	redisClient := redisConnect(redisHost, redisPassword, redisDB)

	// start subscribing to redis cache channel and begin receiving data
	redisClient.Subscribe(redisCacheChannel).Receive()

	domainChannel = make(chan CacheControlMessage)
	go manageCacheChannel(domainChannel, redisClient, redisCacheChannel)

	recordChannel = make(chan CacheControlMessage)
	go manageCacheChannel(recordChannel, redisClient, redisCacheChannel)

	// Start prometheus metrics
	go startPrometheus()

	go func() {
		router := mux.NewRouter().StrictSlash(true)
		router.HandleFunc("/", requestMiddleware(indexView))
		router.HandleFunc("/login", loginView) // No middleware here as its expected to have a clean session state
		router.HandleFunc("/logout", requestMiddleware(logoutView))
		router.HandleFunc("/cache/purge", requestMiddleware(purgeCacheView))
		router.HandleFunc("/cache/record/purge", requestMiddleware(purgeCacheRecordView))
		router.HandleFunc("/domain/create", requestMiddleware(createDomainView))
		router.HandleFunc("/domain/list", requestMiddleware(listDomainView))
		router.HandleFunc("/domain/delete", requestMiddleware(deleteDomainView))
		router.HandleFunc("/record/create", requestMiddleware(createRecordView))
		router.HandleFunc("/record/update", requestMiddleware(updateRecordView))
		router.HandleFunc("/record/list", requestMiddleware(listRecordView))
		router.HandleFunc("/record/list/all", requestMiddleware(listAllRecordView))
		router.HandleFunc("/record/delete", requestMiddleware(deleteRecordView))
		router.HandleFunc("/session/jwt/create", requestMiddleware(createJWTTokenView))
		router.HandleFunc("/user/profile", requestMiddleware(userProfileView))
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", apiPort), router))
	}()

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Fatalf("Signal (%v) received, stopping\n", s)

}
