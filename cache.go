package main

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"

	"github.com/go-redis/redis"
)

// CacheControlMessage -- struct for storing/parsing redis cache control messages
//  					  to the dns server
type CacheControlMessage struct {
	Action string
	Type   string
	Object string
}

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

func addDomainToCache(domain Domain, channel chan<- CacheControlMessage) {
	jsonMSG, err := json.Marshal(domain)
	if err != nil {
		log.Fatal(err)
	}
	msg := CacheControlMessage{
		Action: "create",
		Type:   "domain",
		Object: string(jsonMSG),
	}
	channel <- msg
	return
}

func deleteDomainFromCache(domain Domain, channel chan<- CacheControlMessage) {
	jsonMSG, err := json.Marshal(domain)
	if err != nil {
		log.Fatal(err)
	}
	msg := CacheControlMessage{
		Action: "purge",
		Type:   "domain",
		Object: string(jsonMSG),
	}
	channel <- msg
	return

}

func addRecordToCache(record Record, channel chan<- CacheControlMessage) {
	jsonMSG, err := json.Marshal(record)
	if err != nil {
		log.Fatal(err)
	}
	msg := CacheControlMessage{
		Action: "create",
		Type:   "record",
		Object: string(jsonMSG),
	}
	channel <- msg
	return
}

func deleteRecordFromCache(record Record, channel chan<- CacheControlMessage) {
	jsonMSG, err := json.Marshal(record)
	if err != nil {
		log.Fatal(err)
	}
	msg := CacheControlMessage{
		Action: "purge",
		Type:   "record",
		Object: string(jsonMSG),
	}
	channel <- msg
	return
}

func recordCacheMsgHandler(cacheChannel string, action string, record Record) error {
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return err
	}

	var cacheMsg CacheControlMessage

	// action could be passed directly but sometimes im dumb and forget
	// what action names i set...this helps keep my stupidity from far reach
	switch action {
	case "create", "purge", "update":
		cacheMsg = CacheControlMessage{
			Action: action,
			Type:   "record",
			Object: string(recordJSON),
		}
	default:
		log.Fatalf("Improper action provided to cache handler: %s", action)
	}

	msgJSON, err := json.Marshal(cacheMsg)
	if err != nil {
		return err
	}

	err = redisClient.Publish(cacheChannel, msgJSON).Err()
	if err != nil {
		log.Warning("Unable to publish cache message")
		return err
	}
	return nil

}
