package mytest

import (
	"github.com/go-redis/redis"
	"log"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr:     "152.136.197.135:6379",
		Password: "zhangpeng",
		DB:       0,
	})
	_, err := client.Ping().Result()
	if err != nil {
		log.Printf("Connect redis failed. Error : %v", err)
	}
	log.Printf("init redis success")
}
