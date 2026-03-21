package database

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type Redis struct {
	Client *redis.Client
}

func NewRedis(uri string) (*Redis, error) {
	opt, err := redis.ParseURL(uri)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Redis{Client: client}, nil
}

func (r *Redis) Close() error {
	return r.Client.Close()
}

// Story diff tracking methods
func (r *Redis) AddProcessedID(storyID, contentID string) error {
	ctx := context.Background()
	key := "processed:" + storyID
	return r.Client.SAdd(ctx, key, contentID).Err()
}

func (r *Redis) IsProcessed(storyID, contentID string) (bool, error) {
	ctx := context.Background()
	key := "processed:" + storyID
	return r.Client.SIsMember(ctx, key, contentID).Result()
}

func (r *Redis) SetStoryHash(storyID, hash string) error {
	ctx := context.Background()
	key := "hash:" + storyID
	return r.Client.Set(ctx, key, hash, 24*time.Hour).Err()
}

func (r *Redis) GetStoryHash(storyID string) (string, error) {
	ctx := context.Background()
	key := "hash:" + storyID
	return r.Client.Get(ctx, key).Result()
}

func (r *Redis) CacheStoryContext(storyID string, data interface{}, ttl time.Duration) error {
	ctx := context.Background()
	key := "context:" + storyID
	return r.Client.Set(ctx, key, data, ttl).Err()
}

func (r *Redis) GetCachedStoryContext(storyID string) (string, error) {
	ctx := context.Background()
	key := "context:" + storyID
	return r.Client.Get(ctx, key).Result()
}

// Cache generic JSON data
func (r *Redis) CacheJSON(key string, data string, ttl time.Duration) error {
	ctx := context.Background()
	return r.Client.Set(ctx, key, data, ttl).Err()
}

// GetCachedJSON retrieves cached JSON string
func (r *Redis) GetCachedJSON(key string) (string, error) {
	ctx := context.Background()
	return r.Client.Get(ctx, key).Result()
}

// InvalidateCache deletes a cache key
func (r *Redis) InvalidateCache(key string) error {
	ctx := context.Background()
	return r.Client.Del(ctx, key).Err()
}
