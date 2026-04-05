//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	zcstorage "github.com/alexferl/zerohttp-contrib/storage"
	"github.com/alexferl/zerohttp/middleware/idempotency"
	"github.com/alexferl/zerohttp/zhtest"
	goredis "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func setupRedisStore(t *testing.T) (*goredis.Client, *zcstorage.RedisStorage, func()) {
	t.Helper()

	ctx := context.Background()

	// Start Redis container
	redisContainer, err := redis.Run(ctx, "redis:8-alpine")
	zhtest.AssertNoError(t, err)

	// Get endpoint (host:port format)
	endpoint, err := redisContainer.Endpoint(ctx, "")
	zhtest.AssertNoError(t, err)

	// Create Redis client
	client := goredis.NewClient(&goredis.Options{
		Addr: endpoint,
	})

	// Verify connection
	err = client.Ping(ctx).Err()
	zhtest.AssertNoError(t, err)

	// Create storage with test prefix
	storage := zcstorage.NewRedisStorage(client, zcstorage.RedisStorageConfig{
		KeyPrefix: "test:vinylstore:",
		LockTTL:   30 * time.Second,
	})

	cleanup := func() {
		_ = client.Close()
		_ = redisContainer.Terminate(ctx)
	}

	return client, storage, cleanup
}

func TestRedisStorage_GenericStorage(t *testing.T) {
	_, storage, cleanup := setupRedisStore(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("SetAndGet", func(t *testing.T) {
		key := "testkey:123"
		value := []byte("test-value-data")
		ttl := 1 * time.Hour

		// Set value
		err := storage.Set(ctx, key, value, ttl)
		zhtest.AssertNoError(t, err)

		// Get value
		retrieved, found, err := storage.Get(ctx, key)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, found)
		zhtest.AssertEqual(t, string(value), string(retrieved))
	})

	t.Run("Get_NotFound", func(t *testing.T) {
		_, found, err := storage.Get(ctx, "testkey:nonexistent")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, found)
	})

	t.Run("Delete", func(t *testing.T) {
		key := "testkey:to-delete"
		value := []byte("value-data")

		// Set then delete
		_ = storage.Set(ctx, key, value, 1*time.Hour)
		err := storage.Delete(ctx, key)
		zhtest.AssertNoError(t, err)

		// Verify deletion
		_, found, err := storage.Get(ctx, key)
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, found)
	})

	t.Run("Expiration", func(t *testing.T) {
		key := "testkey:expiring"
		value := []byte("short-lived-value")

		// Set with very short TTL
		err := storage.Set(ctx, key, value, 100*time.Millisecond)
		zhtest.AssertNoError(t, err)

		// Verify value exists
		_, found, err := storage.Get(ctx, key)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, found)

		// Wait for expiration
		time.Sleep(200 * time.Millisecond)

		// Verify value expired
		_, found, err = storage.Get(ctx, key)
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, found)
	})
}

func TestRedisStorage_Idempotency(t *testing.T) {
	_, storage, cleanup := setupRedisStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create idempotency store adapter
	idempAdapter, err := idempotency.NewStorageAdapter(storage)
	zhtest.AssertNoError(t, err)

	t.Run("SetAndGetRecord", func(t *testing.T) {
		key := fmt.Sprintf("idemkey:%d", time.Now().UnixNano())
		record := idempotency.Record{
			StatusCode: 201,
			Body:       []byte(`{"id":"123","status":"created"}`),
		}
		ttl := 24 * time.Hour

		// Set record
		err := idempAdapter.Set(ctx, key, record, ttl)
		zhtest.AssertNoError(t, err)

		// Get record
		retrieved, found, err := idempAdapter.Get(ctx, key)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, found)
		zhtest.AssertEqual(t, record.StatusCode, retrieved.StatusCode)
		zhtest.AssertEqual(t, string(record.Body), string(retrieved.Body))
	})

	t.Run("Get_NotFound", func(t *testing.T) {
		_, found, err := idempAdapter.Get(ctx, "idemkey:nonexistent")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, found)
	})

	t.Run("LockAndUnlock", func(t *testing.T) {
		key := fmt.Sprintf("idemlock:%d", time.Now().UnixNano())

		// Acquire lock
		acquired, err := idempAdapter.Lock(ctx, key)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, acquired)

		// Release lock
		err = idempAdapter.Unlock(ctx, key)
		zhtest.AssertNoError(t, err)
	})

	t.Run("LockPreventsDuplicate", func(t *testing.T) {
		key := fmt.Sprintf("idemlock:dup:%d", time.Now().UnixNano())

		// First lock should succeed
		acquired1, err := idempAdapter.Lock(ctx, key)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, acquired1)
		defer idempAdapter.Unlock(ctx, key)

		// Second lock should fail (lock held)
		acquired2, err := idempAdapter.Lock(ctx, key)
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, acquired2)
	})

	t.Run("RecordExpiration", func(t *testing.T) {
		key := fmt.Sprintf("idemkey:exp:%d", time.Now().UnixNano())
		record := idempotency.Record{
			StatusCode: 200,
			Body:       []byte(`{"status":"ok"}`),
		}

		// Set with short TTL
		err := idempAdapter.Set(ctx, key, record, 100*time.Millisecond)
		zhtest.AssertNoError(t, err)

		// Verify exists
		_, found, err := idempAdapter.Get(ctx, key)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, found)

		// Wait for expiration
		time.Sleep(200 * time.Millisecond)

		// Verify expired
		_, found, err = idempAdapter.Get(ctx, key)
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, found)
	})
}

func TestRedisStorage_RateLimit(t *testing.T) {
	client, _, cleanup := setupRedisStore(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("IncrementAndGet", func(t *testing.T) {
		key := fmt.Sprintf("ratelimit:%d", time.Now().UnixNano())

		// Increment multiple times
		for i := 1; i <= 5; i++ {
			count, err := client.Incr(ctx, key).Result()
			zhtest.AssertNoError(t, err)
			zhtest.AssertEqual(t, int64(i), count)
		}

		// Clean up
		_ = client.Del(ctx, key)
	})

	t.Run("Expire", func(t *testing.T) {
		key := fmt.Sprintf("ratelimit:exp:%d", time.Now().UnixNano())

		// Set with short expiration
		err := client.Set(ctx, key, "1", 100*time.Millisecond).Err()
		zhtest.AssertNoError(t, err)

		// Verify exists
		exists, err := client.Exists(ctx, key).Result()
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, int64(1), exists)

		// Wait for expiration
		time.Sleep(200 * time.Millisecond)

		// Verify expired
		exists, _ = client.Exists(ctx, key).Result()
		zhtest.AssertEqual(t, int64(0), exists)
	})
}

func TestRedisStorage_KeyPrefix(t *testing.T) {
	client, _, cleanup := setupRedisStore(t)
	defer cleanup()

	ctx := context.Background()

	// Keys should have the "test:vinylstore:" prefix
	key := "testkey"
	err := client.Set(ctx, "test:vinylstore:"+key, "value", 1*time.Hour).Err()
	zhtest.AssertNoError(t, err)

	// Scan for keys with prefix
	keys, err := client.Keys(ctx, "test:vinylstore:*").Result()
	zhtest.AssertNoError(t, err)
	zhtest.AssertGreater(t, len(keys), 0)

	// Verify key has correct format
	found := false
	for _, k := range keys {
		if k == "test:vinylstore:"+key {
			found = true
			break
		}
	}
	zhtest.AssertTrue(t, found)
}
