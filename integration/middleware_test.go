//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	zh "github.com/alexferl/zerohttp"
	zcratelimit "github.com/alexferl/zerohttp-contrib/middleware/ratelimit"
	zcstorage "github.com/alexferl/zerohttp-contrib/storage"
	"github.com/alexferl/zerohttp/middleware/idempotency"
	"github.com/alexferl/zerohttp/middleware/ratelimit"
	"github.com/alexferl/zerohttp/zhtest"
	goredis "github.com/redis/go-redis/v9"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

// setupMiddlewareTest creates a test server with middleware
func setupMiddlewareTest(t *testing.T, middlewares ...zh.MiddlewareFunc) (*httptest.Server, *goredis.Client, func()) {
	t.Helper()

	ctx := context.Background()

	// Start Redis container
	redisContainer, err := tcredis.Run(ctx, "redis:8-alpine")
	zhtest.AssertNoError(t, err)

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	zhtest.AssertNoError(t, err)

	// Create Redis client
	redisClient := goredis.NewClient(&goredis.Options{Addr: redisEndpoint})
	err = redisClient.Ping(ctx).Err()
	zhtest.AssertNoError(t, err)

	// Create zerohttp server
	appCfg := zh.Config{
		Server: &http.Server{Addr: "localhost:0"},
	}
	zhServer := zh.New(appCfg)

	// Add middlewares
	zhServer.Use(middlewares...)

	// Add test endpoint
	zhServer.GET("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))

	zhServer.POST("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	}))

	// Create test server using zhServer directly as handler
	ts := httptest.NewServer(zhServer)

	cleanup := func() {
		ts.Close()
		_ = redisClient.Close()
		_ = redisContainer.Terminate(ctx)
	}

	return ts, redisClient, cleanup
}

func TestMiddleware_RateLimit(t *testing.T) {
	ctx := context.Background()

	// Start Redis container
	redisContainer, err := tcredis.Run(ctx, "redis:8-alpine")
	zhtest.AssertNoError(t, err)
	defer redisContainer.Terminate(ctx)

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	zhtest.AssertNoError(t, err)

	redisClient := goredis.NewClient(&goredis.Options{Addr: redisEndpoint})
	defer redisClient.Close()

	// Create rate limit store
	store := zcratelimit.NewRedisStore(redisClient, zcratelimit.RedisStoreConfig{
		Algorithm: ratelimit.TokenBucket,
		Rate:      5,
		Window:    time.Minute,
	})

	// Create rate limit middleware
	cfg := ratelimit.Config{
		Store:        store,
		Rate:         5,
		Window:       time.Minute,
		KeyExtractor: ratelimit.IPKeyExtractor(),
	}

	ts, _, cleanup := setupMiddlewareTest(t, ratelimit.New(cfg))
	defer cleanup()

	// Make requests up to the limit
	for i := 0; i < 5; i++ {
		resp, err := http.Get(ts.URL + "/test")
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	}

	// Next request should be rate limited
	resp, err := http.Get(ts.URL + "/test")
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, http.StatusTooManyRequests, resp.StatusCode)
	resp.Body.Close()
}

func TestMiddleware_RateLimit_ResetWindow(t *testing.T) {
	ctx := context.Background()

	// Start Redis container
	redisContainer, err := tcredis.Run(ctx, "redis:8-alpine")
	zhtest.AssertNoError(t, err)
	defer redisContainer.Terminate(ctx)

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	zhtest.AssertNoError(t, err)

	redisClient := goredis.NewClient(&goredis.Options{Addr: redisEndpoint})
	defer redisClient.Close()

	// Create rate limit store with short window
	store := zcratelimit.NewRedisStore(redisClient, zcratelimit.RedisStoreConfig{
		Algorithm: ratelimit.TokenBucket,
		Rate:      2,
		Window:    100 * time.Millisecond,
	})

	cfg := ratelimit.Config{
		Store:        store,
		Rate:         2,
		Window:       100 * time.Millisecond,
		KeyExtractor: ratelimit.IPKeyExtractor(),
	}

	ts, _, cleanup := setupMiddlewareTest(t, ratelimit.New(cfg))
	defer cleanup()

	// Make requests up to the limit
	for i := 0; i < 2; i++ {
		resp, err := http.Get(ts.URL + "/test")
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	}

	// Next request should be rate limited
	resp, err := http.Get(ts.URL + "/test")
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, http.StatusTooManyRequests, resp.StatusCode)
	resp.Body.Close()

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Should be able to make requests again
	resp, err = http.Get(ts.URL + "/test")
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestMiddleware_Idempotency(t *testing.T) {
	ctx := context.Background()

	// Start Redis container
	redisContainer, err := tcredis.Run(ctx, "redis:8-alpine")
	zhtest.AssertNoError(t, err)
	defer redisContainer.Terminate(ctx)

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	zhtest.AssertNoError(t, err)

	redisClient := goredis.NewClient(&goredis.Options{Addr: redisEndpoint})
	defer redisClient.Close()

	// Setup Redis storage
	redisStorage := zcstorage.NewRedisStorage(redisClient, zcstorage.RedisStorageConfig{
		KeyPrefix: "test:idempotency:",
		LockTTL:   30 * time.Second,
	})

	idempAdapter, err := idempotency.NewStorageAdapter(redisStorage)
	zhtest.AssertNoError(t, err)

	idempCfg := idempotency.Config{
		Store:    idempAdapter,
		Required: true,
		TTL:      24 * time.Hour,
	}

	// Create server with idempotency middleware
	appCfg := zh.Config{
		Server: &http.Server{Addr: "localhost:0"},
	}
	zhServer := zh.New(appCfg)

	requestCount := 0
	zhServer.POST("/create", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":    fmt.Sprintf("item-%d", requestCount),
			"count": requestCount,
		})
	}), idempotency.New(idempCfg))

	ts := httptest.NewServer(zhServer)
	defer ts.Close()

	// First request with idempotency key
	idempotencyKey := "test-key-123"
	body := map[string]string{"name": "Test Item"}
	bodyBytes, _ := json.Marshal(body)

	req1, err := http.NewRequest("POST", ts.URL+"/create", bytes.NewReader(bodyBytes))
	zhtest.AssertNoError(t, err)
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", idempotencyKey)

	resp1, err := http.DefaultClient.Do(req1)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, http.StatusCreated, resp1.StatusCode)

	var resp1Body map[string]any
	json.NewDecoder(resp1.Body).Decode(&resp1Body)
	resp1.Body.Close()

	firstID := resp1Body["id"].(string)
	zhtest.AssertEqual(t, 1, requestCount) // Handler should have been called once

	// Second request with same idempotency key
	req2, err := http.NewRequest("POST", ts.URL+"/create", bytes.NewReader(bodyBytes))
	zhtest.AssertNoError(t, err)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idempotencyKey)

	resp2, err := http.DefaultClient.Do(req2)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, http.StatusCreated, resp2.StatusCode)

	var resp2Body map[string]any
	json.NewDecoder(resp2.Body).Decode(&resp2Body)
	resp2.Body.Close()

	// Should get same response but handler not called again
	zhtest.AssertEqual(t, firstID, resp2Body["id"])
	zhtest.AssertEqual(t, 1, requestCount) // Handler should still have been called only once
}

func TestMiddleware_Idempotency_DifferentKeys(t *testing.T) {
	ctx := context.Background()

	// Start Redis container
	redisContainer, err := tcredis.Run(ctx, "redis:8-alpine")
	zhtest.AssertNoError(t, err)
	defer redisContainer.Terminate(ctx)

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	zhtest.AssertNoError(t, err)

	redisClient := goredis.NewClient(&goredis.Options{Addr: redisEndpoint})
	defer redisClient.Close()

	// Setup Redis storage
	redisStorage := zcstorage.NewRedisStorage(redisClient, zcstorage.RedisStorageConfig{
		KeyPrefix: "test:idempotency:",
		LockTTL:   30 * time.Second,
	})

	idempAdapter, err := idempotency.NewStorageAdapter(redisStorage)
	zhtest.AssertNoError(t, err)

	idempCfg := idempotency.Config{
		Store:    idempAdapter,
		Required: true,
		TTL:      24 * time.Hour,
	}

	// Create server with idempotency middleware
	appCfg := zh.Config{
		Server: &http.Server{Addr: "localhost:0"},
	}
	zhServer := zh.New(appCfg)

	requestCount := 0
	zhServer.POST("/create", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":    fmt.Sprintf("item-%d", requestCount),
			"count": requestCount,
		})
	}), idempotency.New(idempCfg))

	ts := httptest.NewServer(zhServer)
	defer ts.Close()

	body := map[string]string{"name": "Test Item"}
	bodyBytes, _ := json.Marshal(body)

	// First request with idempotency key 1
	req1, _ := http.NewRequest("POST", ts.URL+"/create", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", "key-1")

	resp1, _ := http.DefaultClient.Do(req1)
	var resp1Body map[string]any
	json.NewDecoder(resp1.Body).Decode(&resp1Body)
	resp1.Body.Close()

	// Second request with different idempotency key
	req2, _ := http.NewRequest("POST", ts.URL+"/create", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "key-2")

	resp2, _ := http.DefaultClient.Do(req2)
	var resp2Body map[string]any
	json.NewDecoder(resp2.Body).Decode(&resp2Body)
	resp2.Body.Close()

	// Should have different IDs because different keys
	zhtest.AssertNotEqual(t, resp1Body["id"], resp2Body["id"])
	zhtest.AssertEqual(t, 2, requestCount) // Handler should have been called twice
}

func TestMiddleware_Idempotency_MissingKey(t *testing.T) {
	ctx := context.Background()

	// Start Redis container
	redisContainer, err := tcredis.Run(ctx, "redis:8-alpine")
	zhtest.AssertNoError(t, err)
	defer redisContainer.Terminate(ctx)

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	zhtest.AssertNoError(t, err)

	redisClient := goredis.NewClient(&goredis.Options{Addr: redisEndpoint})
	defer redisClient.Close()

	// Setup Redis storage
	redisStorage := zcstorage.NewRedisStorage(redisClient, zcstorage.RedisStorageConfig{
		KeyPrefix: "test:idempotency:",
		LockTTL:   30 * time.Second,
	})

	idempAdapter, err := idempotency.NewStorageAdapter(redisStorage)
	zhtest.AssertNoError(t, err)

	idempCfg := idempotency.Config{
		Store:    idempAdapter,
		Required: true, // Require key
		TTL:      24 * time.Hour,
	}

	// Create server with idempotency middleware
	appCfg := zh.Config{
		Server: &http.Server{Addr: "localhost:0"},
	}
	zhServer := zh.New(appCfg)

	zhServer.POST("/create", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	}), idempotency.New(idempCfg))

	ts := httptest.NewServer(zhServer)
	defer ts.Close()

	// Request without idempotency key
	body := map[string]string{"name": "Test Item"}
	bodyBytes, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", ts.URL+"/create", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// No Idempotency-Key header

	resp, err := http.DefaultClient.Do(req)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}
