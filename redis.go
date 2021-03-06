package main

import (
	log "github.com/sirupsen/logrus"
	"time"

	"github.com/go-redis/redis"
)

var redisClient *redis.Client
var redisCacheChannelName string

func redisPublish(redisClient *redis.Client, redisChannel string, msg string) bool {
	err := redisClient.Publish(redisChannel, msg).Err()
	if err != nil {
		log.Fatal(err)
		return false
	}
	return true
}

func redisConnect(redisHost string, redisPassword string, redisDB int) *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Password: redisPassword,
		DB:       redisDB,
	})

	// Ping/Pong - (Will be) Used for health check
	go func() {
		for {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			for range ticker.C {
				_, err := redisClient.Ping().Result()
				if err != nil {
					log.Error("[REDIS] Unable to communicate with " + redisHost)
				}
			}

		}
	}()

	return redisClient
}
