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
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"gopkg.in/ini.v1"
)

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
	redisCacheChannelName = cfg.Section("redis").Key("cache_channel").String()

	prometheusPort := cfg.Section("api").Key("prometheus_port").String()
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

	// Start prometheus metrics
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", prometheusPort), nil))
	}()

	go func() {
		router := mux.NewRouter().StrictSlash(true)
		router.HandleFunc("/", indexView)
		router.HandleFunc("/login", loginView)
		router.HandleFunc("/logout", logoutView)
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
