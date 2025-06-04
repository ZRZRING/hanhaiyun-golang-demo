package redis

import (
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
)

type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type Client struct {
	Client *redis.Client
}

func NewRedisClient(cfg Config) (*Client, error) {
	// Build the Redis address
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	// Create a new Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password, // no password set
		DB:       cfg.DB,       // use default DB
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Redis connection established")
	return &Client{Client: client}, nil
}

func (client *Client) Close() error {
	return client.Client.Close()
}
