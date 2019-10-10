package main

import (
	"encoding/json"
	"log"
)

// CacheControlMessage -- struct for storing/parsing redis cache control messages
//  					  to the dns server
type CacheControlMessage struct {
	Action string
	Type   string
	Object string
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
		log.Println("Unable to publish cache message")
		return err
	}
	return nil

}
