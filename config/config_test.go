package config

import (
	"os"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestLoad_Defaults(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set minimal args
	os.Args = []string{"test"}

	cfg, err := Load()
	zhtest.AssertNoError(t, err)

	// Check defaults
	zhtest.AssertEqual(t, "localhost:8080", cfg.Server.BindAddr)
	zhtest.AssertEqual(t, 30*time.Second, cfg.Server.ShutdownTimeout)
	zhtest.AssertEqual(t, "localhost:6379", cfg.Redis.Addr)
	zhtest.AssertEqual(t, "vinylstore:", cfg.Redis.KeyPrefix)
	expectedMongoURI := "mongodb://localhost:27017/?connectTimeoutMS=10000&serverSelectionTimeoutMS=5000&socketTimeoutMS=30000"
	zhtest.AssertEqual(t, expectedMongoURI, cfg.Mongo.URI)
	zhtest.AssertEqual(t, "vinylstore", cfg.Mongo.Database)
	zhtest.AssertEqual(t, "jwt.key", cfg.JWT.PrivateKeyPath)
	zhtest.AssertEqual(t, 15*time.Minute, cfg.JWT.AccessTokenTTL)
	zhtest.AssertEqual(t, 7*24*time.Hour, cfg.JWT.RefreshTokenTTL)
	zhtest.AssertEqual(t, "localhost:4317", cfg.Tracer.Endpoint)
	zhtest.AssertTrue(t, cfg.Tracer.Enabled)
	zhtest.AssertTrue(t, cfg.Tracer.Insecure)
	zhtest.AssertEqual(t, "vinyl-store-api", cfg.App.Name)
	zhtest.AssertTrue(t, cfg.App.SeedData)
	zhtest.AssertTrue(t, cfg.App.CreateAdmin)
	zhtest.AssertEqual(t, "admin@example.com", cfg.App.AdminEmail)
	zhtest.AssertEqual(t, "admin123", cfg.App.AdminPassword)
}

func TestLoad_Flags(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set custom args
	os.Args = []string{
		"test",
		"--bind-addr", "0.0.0.0:9090",
		"--shutdown-timeout", "60s",
		"--redis-addr", "redis:6379",
		"--mongo-uri", "mongodb://mongo:27017",
		"--mongo-database", "testdb",
		"--jwt-access-ttl", "30m",
		"--seed-data=false",
	}

	cfg, err := Load()
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, "0.0.0.0:9090", cfg.Server.BindAddr)
	zhtest.AssertEqual(t, 60*time.Second, cfg.Server.ShutdownTimeout)
	zhtest.AssertEqual(t, "redis:6379", cfg.Redis.Addr)
	zhtest.AssertEqual(t, "mongodb://mongo:27017", cfg.Mongo.URI)
	zhtest.AssertEqual(t, "testdb", cfg.Mongo.Database)
	zhtest.AssertEqual(t, 30*time.Minute, cfg.JWT.AccessTokenTTL)
	zhtest.AssertFalse(t, cfg.App.SeedData)
}

func TestLoad_MultipleFlags(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test various flags
	os.Args = []string{
		"test",
		"--app-name", "custom-app",
		"--tracer-enabled=false",
		"--tracer-endpoint", "jaeger:4317",
		"--create-admin=false",
		"--admin-email", "root@example.com",
	}

	cfg, err := Load()
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, "custom-app", cfg.App.Name)
	zhtest.AssertFalse(t, cfg.Tracer.Enabled)
	zhtest.AssertEqual(t, "jaeger:4317", cfg.Tracer.Endpoint)
	zhtest.AssertFalse(t, cfg.App.CreateAdmin)
	zhtest.AssertEqual(t, "root@example.com", cfg.App.AdminEmail)
}
