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
	"sync"
	"testing"
	"time"

	zh "github.com/alexferl/zerohttp"
	zcstorage "github.com/alexferl/zerohttp-contrib/storage"
	"github.com/alexferl/zerohttp/middleware/idempotency"
	"github.com/alexferl/zerohttp/middleware/jwtauth"
	"github.com/alexferl/zerohttp/zhtest"
	goredis "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/alexferl/zerohttp-example/handlers"
	"github.com/alexferl/zerohttp-example/models"
	"github.com/alexferl/zerohttp-example/store"
)

// setupConcurrencyTest creates a test server for concurrency testing
func setupConcurrencyTest(t *testing.T) (*httptest.Server, *store.MongoStore, func()) {
	t.Helper()

	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := mongodb.Run(ctx, "mongo:8")
	zhtest.AssertNoError(t, err)

	mongoConnStr, err := mongoContainer.ConnectionString(ctx)
	zhtest.AssertNoError(t, err)

	dbName := fmt.Sprintf("testdb_concurrent_%d", time.Now().UnixNano())

	// Create MongoDB store
	mongoStore, err := store.NewMongoStore(mongoConnStr, dbName)
	zhtest.AssertNoError(t, err)

	// Start Redis container
	redisContainer, err := tcredis.Run(ctx, "redis:8-alpine")
	zhtest.AssertNoError(t, err)

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	zhtest.AssertNoError(t, err)

	redisClient := goredis.NewClient(&goredis.Options{Addr: redisEndpoint})
	err = redisClient.Ping(ctx).Err()
	zhtest.AssertNoError(t, err)

	// Setup Redis storage
	redisStorage := zcstorage.NewRedisStorage(redisClient, zcstorage.RedisStorageConfig{
		KeyPrefix: "test:concurrent:",
		LockTTL:   30 * time.Second,
	})

	// Setup JWT config with HS256 for simplicity in tests
	jwtCfg := jwtauth.Config{
		Store:           jwtauth.NewHS256Store([]byte("test-secret-key-at-least-32-bytes!"), jwtauth.HS256Config{Issuer: "test"}),
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	// Setup idempotency config (not required for concurrency tests)
	idempAdapter, err := idempotency.NewStorageAdapter(redisStorage)
	zhtest.AssertNoError(t, err)

	idempCfg := idempotency.Config{
		Store:    idempAdapter,
		Required: false,
		TTL:      24 * time.Hour,
	}

	// Create zerohttp server
	appCfg := zh.Config{
		Server: &http.Server{Addr: "localhost:0"},
	}
	zhServer := zh.New(appCfg)

	// Setup handler
	h := handlers.New(mongoStore, jwtCfg)

	// Setup routes (simplified for concurrency testing)
	zhServer.POST("/users", zh.HandlerFunc(h.CreateUser))
	zhServer.POST("/auth/login", zh.HandlerFunc(h.Login))
	zhServer.POST("/records", zh.HandlerFunc(h.CreateRecord))

	zhServer.Group(func(r zh.Router) {
		r.Use(jwtauth.New(jwtCfg))
		r.POST("/orders", zh.HandlerFunc(h.CreateOrder), idempotency.New(idempCfg))
		r.GET("/records/{id}", zh.HandlerFunc(h.GetRecord))
	})

	ts := httptest.NewServer(zhServer)

	cleanup := func() {
		ts.Close()
		_ = mongoStore.Close(ctx)
		_ = redisClient.Close()
		_ = mongoContainer.Terminate(ctx)
		_ = redisContainer.Terminate(ctx)
	}

	return ts, mongoStore, cleanup
}

// createTestUserAndGetToken creates a user and returns the access token
func createTestUserAndGetToken(t *testing.T, ts *httptest.Server, email, password string) string {
	t.Helper()

	// Create user
	createUserReq := map[string]any{
		"email":    email,
		"name":     "Test User",
		"password": password,
	}
	bodyBytes, _ := json.Marshal(createUserReq)
	resp, err := http.Post(ts.URL+"/users", "application/json", bytes.NewReader(bodyBytes))
	zhtest.AssertNoError(t, err)
	resp.Body.Close()

	// Login
	loginReq := map[string]string{
		"email":    email,
		"password": password,
	}
	bodyBytes, _ = json.Marshal(loginReq)
	resp, err = http.Post(ts.URL+"/auth/login", "application/json", bytes.NewReader(bodyBytes))
	zhtest.AssertNoError(t, err)

	var loginResp map[string]any
	json.NewDecoder(resp.Body).Decode(&loginResp)
	resp.Body.Close()

	return loginResp["access_token"].(string)
}

// createTestRecord creates a record and returns its ID
func createTestRecord(t *testing.T, ts *httptest.Server) string {
	t.Helper()

	createRecordReq := map[string]any{
		"title":     "Test Record",
		"artist":    "Test Artist",
		"year":      2020,
		"format":    "lp",
		"genre":     "jazz",
		"condition": "nm",
		"price":     25.00,
		"stock":     10,
	}
	bodyBytes, _ := json.Marshal(createRecordReq)
	resp, err := http.Post(ts.URL+"/records", "application/json", bytes.NewReader(bodyBytes))
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, http.StatusCreated, resp.StatusCode)

	var recordResp map[string]any
	json.NewDecoder(resp.Body).Decode(&recordResp)
	resp.Body.Close()

	return recordResp["id"].(string)
}

func TestConcurrency_MultipleOrdersStockDeduction(t *testing.T) {
	ts, mongoStore, cleanup := setupConcurrencyTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a record with limited stock
	record := models.NewRecord(models.RecordParams{
		Title:     "Limited Edition",
		Artist:    "Test Artist",
		Format:    models.FormatLP,
		Genre:     models.GenreJazz,
		Condition: models.ConditionNM,
		Price:     50.00,
		Stock:     5, // Only 5 in stock
	})
	err := mongoStore.SaveRecord(ctx, record)
	zhtest.AssertNoError(t, err)

	recordID := record.ID

	// Number of concurrent orders
	numOrders := 10
	var wg sync.WaitGroup
	results := make(chan int, numOrders)

	// Create orders concurrently from different users
	for i := 0; i < numOrders; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Create unique user for each order
			email := fmt.Sprintf("user%d@test.com", index)
			password := "password12345"

			token := createTestUserAndGetToken(t, ts, email, password)

			// Try to create order for 1 item
			createOrderReq := map[string]any{
				"items": []map[string]any{
					{
						"record_id": recordID,
						"quantity":  1,
					},
				},
				"shipping_address": map[string]string{
					"street":   "123 Main St",
					"city":     "SF",
					"state":    "CA",
					"zip_code": "94102",
					"country":  "USA",
				},
			}

			bodyBytes, _ := json.Marshal(createOrderReq)
			req, _ := http.NewRequest("POST", ts.URL+"/orders", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				results <- 0
				return
			}
			resp.Body.Close()

			results <- resp.StatusCode
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)

	// Count successful orders
	successCount := 0
	for status := range results {
		if status == http.StatusCreated {
			successCount++
		}
	}

	// Verify stock was properly deducted (should be 5 successful orders max)
	zhtest.AssertLess(t, successCount, 6) // Should not exceed initial stock

	// Check final stock
	finalRecord, ok, err := mongoStore.GetRecord(ctx, recordID)
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, ok)
	zhtest.AssertEqual(t, 5-successCount, finalRecord.Stock)
}

func TestConcurrency_SameUserMultipleOrders(t *testing.T) {
	ts, mongoStore, cleanup := setupConcurrencyTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a record with sufficient stock
	record := models.NewRecord(models.RecordParams{
		Title:     "Test Record",
		Artist:    "Test Artist",
		Format:    models.FormatLP,
		Genre:     models.GenreRock,
		Condition: models.ConditionNM,
		Price:     30.00,
		Stock:     100,
	})
	err := mongoStore.SaveRecord(ctx, record)
	zhtest.AssertNoError(t, err)

	recordID := record.ID

	// Create a single user
	token := createTestUserAndGetToken(t, ts, "single@test.com", "password12345")

	// Number of concurrent orders from same user
	numOrders := 5
	var wg sync.WaitGroup
	results := make(chan int, numOrders)

	for i := 0; i < numOrders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			createOrderReq := map[string]any{
				"items": []map[string]any{
					{
						"record_id": recordID,
						"quantity":  1,
					},
				},
				"shipping_address": map[string]string{
					"street":   "123 Main St",
					"city":     "SF",
					"state":    "CA",
					"zip_code": "94102",
					"country":  "USA",
				},
			}

			bodyBytes, _ := json.Marshal(createOrderReq)
			req, _ := http.NewRequest("POST", ts.URL+"/orders", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				results <- 0
				return
			}
			resp.Body.Close()

			results <- resp.StatusCode
		}()
	}

	wg.Wait()
	close(results)

	// Count successful orders
	successCount := 0
	for status := range results {
		if status == http.StatusCreated {
			successCount++
		}
	}

	// All orders should succeed (different from stock race condition - this tests same-user ordering)
	zhtest.AssertEqual(t, numOrders, successCount)

	// Verify stock was properly deducted
	finalRecord, ok, err := mongoStore.GetRecord(ctx, recordID)
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, ok)
	zhtest.AssertEqual(t, 100-numOrders, finalRecord.Stock)
}

func TestConcurrency_OrderExceedsStock(t *testing.T) {
	ts, mongoStore, cleanup := setupConcurrencyTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a record with very limited stock
	record := models.NewRecord(models.RecordParams{
		Title:     "Rare Item",
		Artist:    "Test Artist",
		Format:    models.FormatLP,
		Genre:     models.GenreElectronic,
		Condition: models.ConditionMint,
		Price:     100.00,
		Stock:     2, // Only 2 in stock
	})
	err := mongoStore.SaveRecord(ctx, record)
	zhtest.AssertNoError(t, err)

	recordID := record.ID

	// Create user
	token := createTestUserAndGetToken(t, ts, "limited@test.com", "password12345")

	// Try to order more than available stock
	createOrderReq := map[string]any{
		"items": []map[string]any{
			{
				"record_id": recordID,
				"quantity":  5, // Try to order 5 when only 2 available
			},
		},
		"shipping_address": map[string]string{
			"street":   "123 Main St",
			"city":     "SF",
			"state":    "CA",
			"zip_code": "94102",
			"country":  "USA",
		},
	}

	bodyBytes, _ := json.Marshal(createOrderReq)
	req, _ := http.NewRequest("POST", ts.URL+"/orders", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, http.StatusConflict, resp.StatusCode)
	resp.Body.Close()

	// Verify stock unchanged
	finalRecord, ok, err := mongoStore.GetRecord(ctx, recordID)
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, ok)
	zhtest.AssertEqual(t, 2, finalRecord.Stock)
}
