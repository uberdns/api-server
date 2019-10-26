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
	ID        int
	Name      string
	CreatedOn time.Time
}

// Record -- struct for storing information regarding records
type Record struct {
	ID        int
	Name      string
	IP        string
	TTL       int64 //TTL for caching
	CreatedOn time.Time
	DomainID  int
	OwnerID   int
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
)

func main() {
	cfgFile := flag.String("config", "config.ini", "Path to the config file")
	flag.Parse()

	cfg, err := ini.Load(*cfgFile)
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
		router.HandleFunc("/", indexView)
		router.HandleFunc("/login", loginView)
		router.HandleFunc("/logout", logoutView)
		router.HandleFunc("/cache/purge", purgeCacheView)
		router.HandleFunc("/cache/record/purge", purgeCacheRecordView)
		router.HandleFunc("/domain/create", createDomainView)
		router.HandleFunc("/domain/list", listDomainView)
		router.HandleFunc("/domain/delete", deleteDomainView)
		router.HandleFunc("/record/create", createRecordView)
		router.HandleFunc("/record/update", updateRecordView)
		router.HandleFunc("/record/list", listRecordView)
		router.HandleFunc("/record/delete", deleteRecordView)
		router.HandleFunc("/session/jwt/create", createJWTTokenView)
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", apiPort), router))
	}()

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Fatalf("Signal (%v) received, stopping\n", s)

}
