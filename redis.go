package main

import "github.com/go-redis/redis"

var redisClient *redis.Client
var redisCacheChannelName string
