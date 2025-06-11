package redis

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	AnnouncementKeyLabel = "announcement::"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient() (*RedisClient, error) {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	addr := host + ":" + port
	//password := os.Getenv("REDIS_PASSWORD")
	fmt.Println("Establish connection to: " + addr)
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		//Password:  password,
		DB: 0,
		//TLSConfig: &tls.Config{},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Err(err).Msg(err.Error())
		return nil, err
	} else {
		log.Info().Msg("Redis connection created successfully")
	}

	return &RedisClient{client: rdb}, nil
}

func (rc *RedisClient) GetClient() *redis.Client {
	return rc.client
}
