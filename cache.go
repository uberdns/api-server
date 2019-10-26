package main

import (
	"encoding/json"
	"log"

	"github.com/go-redis/redis"
)

func manageCacheChannel(channel <-chan CacheControlMessage, redisClient *redis.Client, redisChannel string) {
	for {
		select {
		case msg := <-channel:
			msgJSON, err := json.Marshal(msg)
			if err != nil {
				log.Fatal(err)
			}
			if !redisPublish(redisClient, redisChannel, string(msgJSON)) {
				log.Fatalf("Could not publish message to Redis")
			}
		}
	}
}
