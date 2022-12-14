package cache

import (
	"context"
	"golang-test-task/internal/entities"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mailru/easyjson"
)

// RedisClient is for wrapping original redis.Client
type RedisClient struct {
	client     *redis.Client
	maxRetries int
	duration   time.Duration
}

// NewRedisClientForTest is constructor for testing
func NewRedisClientForTest(client *redis.Client) *RedisClient {
	return &RedisClient{client: client, maxRetries: 10, duration: 5 * time.Minute}
}

// NewRedisClient is constructor for RedisClient
func NewRedisClient(ctx context.Context, config RedisConfig) *RedisClient {
	options := redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	}
	client := redis.NewClient(&options)

	if _, err := client.Ping(ctx).Result(); err != nil {
		panic(err)
	}
	maxRetries := 10
	duration := 5 * time.Minute
	return &RedisClient{client: client, maxRetries: maxRetries, duration: duration}
}

// FindItemValue gives cached item
func (r *RedisClient) FindItemValue(ctx context.Context, key string) (*entities.APIAdItem, error) {
	v, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(v) == 0 {
		return nil, nil
	}
	var itm entities.APIAdItem
	err = easyjson.Unmarshal([]byte(v), &itm)
	if err != nil {
		return nil, err
	}
	return &itm, nil
}

// SetItemValue setting item to cache
func (r *RedisClient) SetItemValue(ctx context.Context, key string, item entities.APIAdItem) error {
	bs, err := easyjson.Marshal(item)
	if err != nil {
		return err
	}
	_, err = r.client.Set(ctx, key, string(bs), r.duration).Result()
	return err
}

// GetDuration gives duration of key
func (r *RedisClient) GetDuration() time.Duration {
	return r.duration
}
