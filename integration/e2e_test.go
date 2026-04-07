//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	zh "github.com/alexferl/zerohttp"
	zcjwtauth "github.com/alexferl/zerohttp-contrib/middleware/jwtauth"
	zcstorage "github.com/alexferl/zerohttp-contrib/storage"
	"github.com/alexferl/zerohttp/middleware/idempotency"
	"github.com/alexferl/zerohttp/middleware/jwtauth"
	"github.com/alexferl/zerohttp/zhtest"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	goredis "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/alexferl/zerohttp-example/handlers"
	"github.com/alexferl/zerohttp-example/store"
)

// testServer holds the test server and its dependencies
type testServer struct {
	Server      *httptest.Server
	MongoStore  *store.MongoStore
	RedisClient *goredis.Client
}

// setupTestServer creates a full test server with real MongoDB and Redis
func setupTestServer(t *testing.T) *testServer {
	t.Helper()

	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := mongodb.Run(ctx, "mongo:8")
	zhtest.AssertNoError(t, err)

	mongoConnStr, err := mongoContainer.ConnectionString(ctx)
	zhtest.AssertNoError(t, err)

	dbName := fmt.Sprintf("testdb_%d", time.Now().UnixNano())

	// Create MongoDB store
	mongoStore, err := store.NewMongoStore(mongoConnStr, dbName)
	zhtest.AssertNoError(t, err)

	// Start Redis container
	redisContainer, err := tcredis.Run(ctx, "redis:8-alpine")
	zhtest.AssertNoError(t, err)

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	zhtest.AssertNoError(t, err)

	// Create Redis client
	redisClient := goredis.NewClient(&goredis.Options{Addr: redisEndpoint})
	err = redisClient.Ping(ctx).Err()
	zhtest.AssertNoError(t, err)

	// Setup Redis storage
	redisStorage := zcstorage.NewRedisStorage(redisClient, zcstorage.RedisStorageConfig{
		KeyPrefix: "test:e2e:",
		LockTTL:   30 * time.Second,
	})

	// Generate ECDSA key for JWT
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	zhtest.AssertNoError(t, err)

	key, err := jwk.Import(privateKey)
	zhtest.AssertNoError(t, err)

	keySet := jwk.NewSet()
	err = keySet.AddKey(key)
	zhtest.AssertNoError(t, err)

	// Setup JWT config
	zcfg := zcjwtauth.Config{
		KeySet:    keySet,
		Algorithm: jwa.ES256(),
		Storage:   redisStorage,
	}
	jwtCfg := jwtauth.Config{
		Store:           zcjwtauth.NewTokenStore(zcfg),
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	// Setup idempotency config
	idempAdapter, err := idempotency.NewStorageAdapter(redisStorage)
	zhtest.AssertNoError(t, err)
	idempCfg := idempotency.Config{
		Store:    idempAdapter,
		Required: false, // Don't require idempotency in tests
		TTL:      24 * time.Hour,
	}

	// Create zerohttp server
	appCfg := zh.Config{
		Server: &http.Server{Addr: "localhost:0"},
	}
	zhServer := zh.New(appCfg)

	// Setup handler
	h := handlers.New(mongoStore, jwtCfg)

	// Setup routes
	setupTestRoutes(zhServer, h, jwtCfg, idempCfg)

	// Create test server using zhServer directly as handler (it implements http.Handler via embedded Router)
	ts := httptest.NewServer(zhServer)

	t.Cleanup(func() {
		ts.Close()
		_ = mongoStore.Close(ctx)
		_ = redisClient.Close()
		_ = mongoContainer.Terminate(ctx)
		_ = redisContainer.Terminate(ctx)
	})

	return &testServer{
		Server:      ts,
		MongoStore:  mongoStore,
		RedisClient: redisClient,
	}
}

// setupTestRoutes sets up routes for e2e testing
func setupTestRoutes(zhServer *zh.Server, h *handlers.Handler, jwtCfg jwtauth.Config, idempCfg idempotency.Config) {
	// Public routes
	zhServer.POST("/auth/login", zh.HandlerFunc(h.Login))
	zhServer.POST("/users", zh.HandlerFunc(h.CreateUser))
	zhServer.GET("/records", zh.HandlerFunc(h.ListRecords))
	zhServer.GET("/records/{id}", zh.HandlerFunc(h.GetRecord))

	// Token refresh
	zhServer.POST("/auth/refresh", jwtauth.RefreshTokenHandler(jwtCfg))

	// Public auth endpoints (don't require valid access token)
	zhServer.POST("/auth/logout", jwtauth.LogoutTokenHandler(jwtCfg))

	// Protected routes
	zhServer.Group(func(r zh.Router) {
		r.Use(jwtauth.New(jwtCfg))

		r.GET("/users/{id}", zh.HandlerFunc(h.GetUser))
		r.PATCH("/users/{id}", zh.HandlerFunc(h.UpdateUser))
		r.POST("/users/{id}/deactivate", zh.HandlerFunc(h.DeactivateUser))
		r.POST("/orders", zh.HandlerFunc(h.CreateOrder), idempotency.New(idempCfg))
		r.GET("/orders", zh.HandlerFunc(h.ListOrders))
		r.GET("/orders/{id}", zh.HandlerFunc(h.GetOrder))
		r.POST("/orders/{id}/cancel", zh.HandlerFunc(h.CancelOrder))
		r.GET("/users", zh.HandlerFunc(h.ListUsers))
		r.POST("/records", zh.HandlerFunc(h.CreateRecord))
		r.PATCH("/records/{id}", zh.HandlerFunc(h.UpdateRecord))
		r.POST("/records/{id}/archive", zh.HandlerFunc(h.ArchiveRecord))
		r.GET("/inventory", zh.HandlerFunc(h.ListInventory))
		r.POST("/inventory/{record_id}/restock", zh.HandlerFunc(h.RestockRecord))
	})
}

// makeRequest helper makes HTTP requests to the test server
func makeRequest(t *testing.T, ts *testServer, method, path string, body any, authToken string) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		zhtest.AssertNoError(t, err)
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, ts.Server.URL+path, bodyReader)
	zhtest.AssertNoError(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := http.DefaultClient.Do(req)
	zhtest.AssertNoError(t, err)

	return resp
}

func TestE2E_UserRegistrationAndLogin(t *testing.T) {
	ts := setupTestServer(t)

	// Register a new user
	createUserReq := map[string]any{
		"email":    "test@example.com",
		"name":     "Test User",
		"password": "securepassword123",
	}

	resp := makeRequest(t, ts, "POST", "/users", createUserReq, "")
	zhtest.AssertEqual(t, http.StatusCreated, resp.StatusCode)

	var userResp map[string]any
	err := json.NewDecoder(resp.Body).Decode(&userResp)
	zhtest.AssertNoError(t, err)
	resp.Body.Close()

	zhtest.AssertNotEmpty(t, userResp["id"])
	zhtest.AssertEqual(t, "test@example.com", userResp["email"])
	zhtest.AssertEqual(t, "Test User", userResp["name"])

	// Login with the created user
	loginReq := map[string]string{
		"email":    "test@example.com",
		"password": "securepassword123",
	}

	resp = makeRequest(t, ts, "POST", "/auth/login", loginReq, "")
	zhtest.AssertEqual(t, http.StatusOK, resp.StatusCode)

	var loginResp map[string]any
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	zhtest.AssertNoError(t, err)
	resp.Body.Close()

	zhtest.AssertNotEmpty(t, loginResp["access_token"])
	zhtest.AssertNotEmpty(t, loginResp["refresh_token"])
	zhtest.AssertEqual(t, "Bearer", loginResp["token_type"])
}

func TestE2E_CreateAndGetRecord(t *testing.T) {
	ts := setupTestServer(t)

	// Create a user first (required for auth)
	createUserReq := map[string]any{
		"email":    "recordtest@example.com",
		"name":     "Record Test User",
		"password": "securepassword123",
	}

	resp := makeRequest(t, ts, "POST", "/users", createUserReq, "")
	zhtest.AssertEqual(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Login to get token
	loginReq := map[string]string{
		"email":    "recordtest@example.com",
		"password": "securepassword123",
	}

	resp = makeRequest(t, ts, "POST", "/auth/login", loginReq, "")
	zhtest.AssertEqual(t, http.StatusOK, resp.StatusCode)

	var loginResp map[string]any
	err := json.NewDecoder(resp.Body).Decode(&loginResp)
	zhtest.AssertNoError(t, err)
	resp.Body.Close()

	accessToken := loginResp["access_token"].(string)

	// Create a record (requires auth)
	createRecordReq := map[string]any{
		"title":     "Kind of Blue",
		"artist":    "Miles Davis",
		"year":      1959,
		"format":    "lp",
		"genre":     "jazz",
		"condition": "nm",
		"price":     29.99,
		"stock":     5,
	}

	resp = makeRequest(t, ts, "POST", "/records", createRecordReq, accessToken)
	zhtest.AssertEqual(t, http.StatusCreated, resp.StatusCode)

	var recordResp map[string]any
	err = json.NewDecoder(resp.Body).Decode(&recordResp)
	zhtest.AssertNoError(t, err)
	resp.Body.Close()

	recordID := recordResp["id"].(string)
	zhtest.AssertNotEmpty(t, recordID)
	zhtest.AssertEqual(t, "Kind of Blue", recordResp["title"])

	// Get the record (public endpoint)
	resp = makeRequest(t, ts, "GET", "/records/"+recordID, nil, "")
	zhtest.AssertEqual(t, http.StatusOK, resp.StatusCode)

	var getResp map[string]any
	err = json.NewDecoder(resp.Body).Decode(&getResp)
	zhtest.AssertNoError(t, err)
	resp.Body.Close()

	zhtest.AssertEqual(t, recordID, getResp["id"])
	zhtest.AssertEqual(t, "Kind of Blue", getResp["title"])
}

func TestE2E_OrderFlow(t *testing.T) {
	ts := setupTestServer(t)

	// Create a user first
	createUserReq := map[string]any{
		"email":    "buyer@example.com",
		"name":     "Buyer User",
		"password": "password12345",
	}

	resp := makeRequest(t, ts, "POST", "/users", createUserReq, "")
	zhtest.AssertEqual(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Login to get token
	loginReq := map[string]string{
		"email":    "buyer@example.com",
		"password": "password12345",
	}

	resp = makeRequest(t, ts, "POST", "/auth/login", loginReq, "")
	zhtest.AssertEqual(t, http.StatusOK, resp.StatusCode)

	var loginResp map[string]any
	err := json.NewDecoder(resp.Body).Decode(&loginResp)
	zhtest.AssertNoError(t, err)
	resp.Body.Close()

	accessToken := loginResp["access_token"].(string)

	// Create a record
	createRecordReq := map[string]any{
		"title":     "Abbey Road",
		"artist":    "The Beatles",
		"year":      1969,
		"format":    "lp",
		"genre":     "rock",
		"condition": "nm",
		"price":     25.00,
		"stock":     10,
	}

	resp = makeRequest(t, ts, "POST", "/records", createRecordReq, accessToken)
	zhtest.AssertEqual(t, http.StatusCreated, resp.StatusCode)

	var recordResp map[string]any
	err = json.NewDecoder(resp.Body).Decode(&recordResp)
	zhtest.AssertNoError(t, err)
	resp.Body.Close()

	recordID := recordResp["id"].(string)

	// Create an order
	createOrderReq := map[string]any{
		"items": []map[string]any{
			{
				"record_id": recordID,
				"quantity":  2,
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

	resp = makeRequest(t, ts, "POST", "/orders", createOrderReq, accessToken)
	zhtest.AssertEqual(t, http.StatusCreated, resp.StatusCode)

	var orderResp map[string]any
	err = json.NewDecoder(resp.Body).Decode(&orderResp)
	zhtest.AssertNoError(t, err)
	resp.Body.Close()

	orderID := orderResp["id"].(string)
	zhtest.AssertNotEmpty(t, orderID)

	// Get the order
	resp = makeRequest(t, ts, "GET", "/orders/"+orderID, nil, accessToken)
	zhtest.AssertEqual(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestE2E_ListRecords(t *testing.T) {
	ts := setupTestServer(t)

	// Create a user first
	createUserReq := map[string]any{
		"email":    "listtest@example.com",
		"name":     "List Test User",
		"password": "securepassword123",
	}

	resp := makeRequest(t, ts, "POST", "/users", createUserReq, "")
	zhtest.AssertEqual(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Login to get token
	loginReq := map[string]string{
		"email":    "listtest@example.com",
		"password": "securepassword123",
	}

	resp = makeRequest(t, ts, "POST", "/auth/login", loginReq, "")
	zhtest.AssertEqual(t, http.StatusOK, resp.StatusCode)

	var loginResp map[string]any
	err := json.NewDecoder(resp.Body).Decode(&loginResp)
	zhtest.AssertNoError(t, err)
	resp.Body.Close()

	accessToken := loginResp["access_token"].(string)

	// Create multiple records (requires auth)
	records := []map[string]any{
		{
			"title":     "Record A",
			"artist":    "Artist 1",
			"year":      2020,
			"format":    "lp",
			"genre":     "jazz",
			"condition": "nm",
			"price":     20.00,
			"stock":     5,
		},
		{
			"title":     "Record B",
			"artist":    "Artist 2",
			"year":      2021,
			"format":    "lp",
			"genre":     "rock",
			"condition": "vg+",
			"price":     25.00,
			"stock":     5,
		},
	}

	for _, record := range records {
		resp := makeRequest(t, ts, "POST", "/records", record, accessToken)
		zhtest.AssertEqual(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()
	}

	// List records (public endpoint)
	resp = makeRequest(t, ts, "GET", "/records", nil, "")
	zhtest.AssertEqual(t, http.StatusOK, resp.StatusCode)

	var listResp map[string]any
	err = json.NewDecoder(resp.Body).Decode(&listResp)
	zhtest.AssertNoError(t, err)
	resp.Body.Close()

	items := listResp["items"].([]any)
	zhtest.AssertGreater(t, len(items), 0)
}
