package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	App       AppConfig
	Server    ServerConfig
	Redis     RedisConfig
	Mongo     MongoConfig
	JWT       JWTConfig
	Tracer    TracerConfig
	RateLimit RateLimitConfig
}

// AppConfig holds general application configuration
type AppConfig struct {
	Name          string `mapstructure:"app_name"`
	SeedData      bool   `mapstructure:"seed_data"`
	CreateAdmin   bool   `mapstructure:"create_admin"`
	AdminEmail    string `mapstructure:"admin_email"`
	AdminPassword string `mapstructure:"admin_password"`
	EnablePprof   bool   `mapstructure:"enable_pprof"`
	PprofUsername string `mapstructure:"pprof_username"`
	PprofPassword string `mapstructure:"pprof_password"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	BindAddr        string        `mapstructure:"bind_addr"`
	BindTLSAddr     string        `mapstructure:"bind_tls_addr"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	CertFile        string        `mapstructure:"cert_file"`
	KeyFile         string        `mapstructure:"key_file"`
	EnableHSTS      bool          `mapstructure:"enable_hsts"`
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Addr      string        `mapstructure:"redis_addr"`
	KeyPrefix string        `mapstructure:"redis_key_prefix"`
	LockTTL   time.Duration `mapstructure:"redis_lock_ttl"`
}

// MongoConfig holds MongoDB connection configuration
type MongoConfig struct {
	URI      string `mapstructure:"mongo_uri"`
	Database string `mapstructure:"mongo_database"`
}

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	PrivateKeyPath  string        `mapstructure:"jwt_private_key_path"`
	AccessTokenTTL  time.Duration `mapstructure:"jwt_access_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"jwt_refresh_ttl"`
}

// TracerConfig holds OpenTelemetry tracer configuration
type TracerConfig struct {
	Enabled  bool   `mapstructure:"tracer_enabled"`
	Endpoint string `mapstructure:"tracer_endpoint"`
	Insecure bool   `mapstructure:"tracer_insecure"`
}

// TierRateLimitConfig holds rate limiting configuration for a specific tier
type TierRateLimitConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	Rate    int           `mapstructure:"rate"`
	Window  time.Duration `mapstructure:"window"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled bool `mapstructure:"rate_limit_enabled"`

	// Public tier - for unauthenticated/public endpoints (keyed by IP)
	Public TierRateLimitConfig `mapstructure:"public"`

	// Authenticated tier - for API users with valid JWT (keyed by user ID)
	Authenticated TierRateLimitConfig `mapstructure:"authenticated"`

	// AuthEndpoints tier - strict rate limit for login/register (keyed by IP)
	AuthEndpoints TierRateLimitConfig `mapstructure:"auth_endpoints"`
}

// Load reads configuration from CLI flags, environment variables, and config files
// Priority: CLI flags > env vars > config file > defaults
func Load() (*Config, error) {
	// Create config with defaults
	cfg := &Config{
		App: AppConfig{
			Name:          "vinyl-store-api",
			SeedData:      true,
			CreateAdmin:   true,
			AdminEmail:    "admin@example.com",
			AdminPassword: "admin123",
			EnablePprof:   false,
		},
		Server: ServerConfig{
			BindAddr:        "localhost:8080",
			ShutdownTimeout: 30 * time.Second,
		},
		Redis: RedisConfig{
			Addr:      "localhost:6379",
			KeyPrefix: "vinylstore:",
			LockTTL:   30 * time.Second,
		},
		Mongo: MongoConfig{
			URI:      "mongodb://localhost:27017/?connectTimeoutMS=10000&serverSelectionTimeoutMS=5000&socketTimeoutMS=30000",
			Database: "vinylstore",
		},
		JWT: JWTConfig{
			PrivateKeyPath:  "jwt.key",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
		Tracer: TracerConfig{
			Enabled:  true,
			Endpoint: "localhost:4317",
			Insecure: true,
		},
		RateLimit: RateLimitConfig{
			Enabled: true,
			Public: TierRateLimitConfig{
				Enabled: true,
				Rate:    30,
				Window:  time.Minute,
			},
			Authenticated: TierRateLimitConfig{
				Enabled: true,
				Rate:    100,
				Window:  time.Minute,
			},
			AuthEndpoints: TierRateLimitConfig{
				Enabled: true,
				Rate:    5,
				Window:  time.Minute,
			},
		},
	}

	// Define CLI flags bound to struct fields
	fs := pflag.NewFlagSet("vinylstore", pflag.ContinueOnError)
	fs.StringVar(&cfg.App.Name, "app-name", cfg.App.Name, "Application name")
	fs.BoolVar(&cfg.App.SeedData, "seed-data", cfg.App.SeedData, "Seed sample data")
	fs.BoolVar(&cfg.App.CreateAdmin, "create-admin", cfg.App.CreateAdmin, "Create default admin user")
	fs.StringVar(&cfg.App.AdminEmail, "admin-email", cfg.App.AdminEmail, "Admin email")
	fs.StringVar(&cfg.App.AdminPassword, "admin-password", cfg.App.AdminPassword, "Admin password")
	fs.BoolVar(&cfg.App.EnablePprof, "enable-pprof", cfg.App.EnablePprof, "Enable pprof endpoints")
	fs.StringVar(&cfg.App.PprofUsername, "pprof-username", cfg.App.PprofUsername, "pprof basic auth username (default: pprof)")
	fs.StringVar(&cfg.App.PprofPassword, "pprof-password", cfg.App.PprofPassword, "pprof basic auth password (default: auto-generated)")

	fs.StringVar(&cfg.Server.BindAddr, "bind-addr", cfg.Server.BindAddr, "Server address")
	fs.StringVar(&cfg.Server.BindTLSAddr, "bind-tls-addr", cfg.Server.BindTLSAddr, "Server TLS address")
	fs.DurationVar(&cfg.Server.ShutdownTimeout, "shutdown-timeout", cfg.Server.ShutdownTimeout, "Shutdown timeout")
	fs.StringVar(&cfg.Server.CertFile, "cert-file", cfg.Server.CertFile, "Path to TLS certificate file")
	fs.StringVar(&cfg.Server.KeyFile, "key-file", cfg.Server.KeyFile, "Path to TLS key file")
	fs.BoolVar(&cfg.Server.EnableHSTS, "enable-hsts", cfg.Server.EnableHSTS, "Enable HSTS (HTTP Strict Transport Security) header")

	fs.StringVar(&cfg.Redis.Addr, "redis-addr", cfg.Redis.Addr, "Redis address")
	fs.StringVar(&cfg.Redis.KeyPrefix, "redis-key-prefix", cfg.Redis.KeyPrefix, "Redis key prefix")
	fs.DurationVar(&cfg.Redis.LockTTL, "redis-lock-ttl", cfg.Redis.LockTTL, "Redis lock TTL")

	fs.StringVar(&cfg.Mongo.URI, "mongo-uri", cfg.Mongo.URI, "MongoDB URI")
	fs.StringVar(&cfg.Mongo.Database, "mongo-database", cfg.Mongo.Database, "MongoDB database name")

	fs.StringVar(&cfg.JWT.PrivateKeyPath, "jwt-private-key-path", cfg.JWT.PrivateKeyPath, "Path to JWT ECDSA private key file")
	fs.DurationVar(&cfg.JWT.AccessTokenTTL, "jwt-access-ttl", cfg.JWT.AccessTokenTTL, "JWT access token TTL")
	fs.DurationVar(&cfg.JWT.RefreshTokenTTL, "jwt-refresh-ttl", cfg.JWT.RefreshTokenTTL, "JWT refresh token TTL")

	fs.BoolVar(&cfg.Tracer.Enabled, "tracer-enabled", cfg.Tracer.Enabled, "Enable OpenTelemetry tracer")
	fs.StringVar(&cfg.Tracer.Endpoint, "tracer-endpoint", cfg.Tracer.Endpoint, "Tracer endpoint")
	fs.BoolVar(&cfg.Tracer.Insecure, "tracer-insecure", cfg.Tracer.Insecure, "Use insecure tracer connection")

	fs.BoolVar(&cfg.RateLimit.Enabled, "rate-limit-enabled", cfg.RateLimit.Enabled, "Enable tiered rate limiting")
	fs.IntVar(&cfg.RateLimit.Public.Rate, "rate-limit-public-rate", cfg.RateLimit.Public.Rate, "Public tier: requests per window")
	fs.DurationVar(&cfg.RateLimit.Public.Window, "rate-limit-public-window", cfg.RateLimit.Public.Window, "Public tier: time window")
	fs.IntVar(&cfg.RateLimit.Authenticated.Rate, "rate-limit-auth-rate", cfg.RateLimit.Authenticated.Rate, "Authenticated tier: requests per window")
	fs.DurationVar(&cfg.RateLimit.Authenticated.Window, "rate-limit-auth-window", cfg.RateLimit.Authenticated.Window, "Authenticated tier: time window")
	fs.IntVar(&cfg.RateLimit.AuthEndpoints.Rate, "rate-limit-login-rate", cfg.RateLimit.AuthEndpoints.Rate, "Auth endpoints tier: requests per window")
	fs.DurationVar(&cfg.RateLimit.AuthEndpoints.Window, "rate-limit-login-window", cfg.RateLimit.AuthEndpoints.Window, "Auth endpoints tier: time window")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	v := viper.New()
	v.SetEnvPrefix("VINYL")
	v.AutomaticEnv()

	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/vinylstore/")
	v.AddConfigPath("$HOME/.config/vinylstore")

	_ = v.ReadInConfig()

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}
