package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	zcjwtauth "github.com/alexferl/zerohttp-contrib/middleware/jwtauth"
	zcratelimit "github.com/alexferl/zerohttp-contrib/middleware/ratelimit"
	zctracer "github.com/alexferl/zerohttp-contrib/middleware/tracer"
	zcstorage "github.com/alexferl/zerohttp-contrib/storage"
	"github.com/alexferl/zerohttp/healthcheck"
	zl "github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/middleware/compress"
	"github.com/alexferl/zerohttp/middleware/idempotency"
	"github.com/alexferl/zerohttp/middleware/jwtauth"
	"github.com/alexferl/zerohttp/middleware/mediatype"
	"github.com/alexferl/zerohttp/middleware/ratelimit"
	"github.com/alexferl/zerohttp/middleware/tracer"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/redis/go-redis/v9"

	"github.com/alexferl/zerohttp-example/config"
	"github.com/alexferl/zerohttp-example/handlers"
	"github.com/alexferl/zerohttp-example/store"
)

func setupRoutes(app *zh.Server, h *handlers.Handler, jwtCfg jwtauth.Config, idempotencyCfg idempotency.Config, cfg *config.Config, redisClient *redis.Client) {
	// Public routes
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message": "Welcome to the Vinyl Store API!",
			"version": "v1",
		})
	}))
	app.POST("/auth/login", zh.HandlerFunc(h.Login))
	app.POST("/users", zh.HandlerFunc(h.CreateUser))
	app.GET("/records", zh.HandlerFunc(h.ListRecords))
	app.GET("/records/{id}", zh.HandlerFunc(h.GetRecord))
	app.GET("/inventory", zh.HandlerFunc(h.ListInventory))

	// Token refresh (public but validates refresh token)
	app.POST("/auth/refresh", jwtauth.RefreshTokenHandler(jwtCfg))

	// Protected routes group - JWT middleware only applies to routes inside
	app.Group(func(r zh.Router) {
		r.Use(
			jwtauth.New(jwtCfg),
			setupAuthenticatedRateLimit(cfg, redisClient, app),
		)

		r.POST("/auth/logout", jwtauth.LogoutTokenHandler(jwtCfg))

		// User routes (owner only)
		r.Group(func(r2 zh.Router) {
			r2.Use(handlers.RequireOwner())
			r2.GET("/users/{id}", zh.HandlerFunc(h.GetUser))
			r2.PATCH("/users/{id}", zh.HandlerFunc(h.UpdateUser))
			r2.POST("/users/{id}/deactivate", zh.HandlerFunc(h.DeactivateUser))
		})

		// Order routes (POST uses idempotency)
		r.POST("/orders", zh.HandlerFunc(h.CreateOrder), idempotency.New(idempotencyCfg))
		r.GET("/orders", zh.HandlerFunc(h.ListOrders))
		r.GET("/orders/{id}", zh.HandlerFunc(h.GetOrder))
		r.POST("/orders/{id}/cancel", zh.HandlerFunc(h.CancelOrder))

		// Admin routes
		r.Group(func(r2 zh.Router) {
			r2.Use(handlers.RequireAdmin())
			r2.GET("/users", zh.HandlerFunc(h.ListUsers))
			r2.POST("/records", zh.HandlerFunc(h.CreateRecord))
			r2.PATCH("/records/{id}", zh.HandlerFunc(h.UpdateRecord))
			r2.POST("/records/{id}/archive", zh.HandlerFunc(h.ArchiveRecord))
			r2.POST("/inventory/{record_id}/restock", zh.HandlerFunc(h.RestockRecord))
		})
	})
}

func setupTracer(ctx context.Context, cfg *config.Config) (*zctracer.OTelTracer, func()) {
	if !cfg.Tracer.Enabled {
		return nil, func() {}
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tracerImpl, shutdown, err := zctracer.NewGRPCDefault(ctx, cfg.App.Name, cfg.Tracer.Endpoint, cfg.Tracer.Insecure)
	if err != nil {
		zl.GetGlobalLogger().Fatal("Failed to create tracer", zl.E(err))
	}
	return tracerImpl, shutdown
}

func setupRedisStorage(ctx context.Context, cfg config.RedisConfig) (*redis.Client, *zcstorage.RedisStorage) {
	client := redis.NewClient(&redis.Options{Addr: cfg.Addr})
	if err := client.Ping(ctx).Err(); err != nil {
		zl.GetGlobalLogger().Fatal("Failed to connect to Redis", zl.E(err))
	}

	storage := zcstorage.NewRedisStorage(client, zcstorage.RedisStorageConfig{
		KeyPrefix: cfg.KeyPrefix,
		LockTTL:   cfg.LockTTL,
	})
	return client, storage
}

func setupJWTConfig(redisStorage *zcstorage.RedisStorage, cfg *config.Config) (jwtauth.Config, error) {
	keySet, err := loadOrCreateKey(cfg.JWT.PrivateKeyPath)
	if err != nil {
		return jwtauth.Config{}, err
	}

	zcfg := zcjwtauth.Config{
		KeySet:    keySet,
		Algorithm: jwa.ES256(),
		Storage:   redisStorage,
	}

	return jwtauth.Config{
		Store:           zcjwtauth.NewTokenStore(zcfg),
		RequiredClaims:  []string{"sub"},
		AccessTokenTTL:  cfg.JWT.AccessTokenTTL,
		RefreshTokenTTL: cfg.JWT.RefreshTokenTTL,
	}, nil
}

func setupIdempotencyConfig(redisStorage *zcstorage.RedisStorage) idempotency.Config {
	storeAdapter, err := idempotency.NewStorageAdapter(redisStorage)
	if err != nil {
		zl.GetGlobalLogger().Fatal("Failed to create idempotency store", zl.E(err))
	}

	return idempotency.Config{
		Store:    storeAdapter,
		Required: true,
		TTL:      24 * time.Hour,
	}
}

func setupHealthcheck(app *zh.Server, redisClient *redis.Client, mongoStore *store.MongoStore) {
	checkDependencies := func(w http.ResponseWriter, r *http.Request) error {
		if err := redisClient.Ping(r.Context()).Err(); err != nil {
			return zh.NewProblemDetail(http.StatusServiceUnavailable, "Redis not available").Render(w)
		}
		if err := mongoStore.Ping(r.Context()); err != nil {
			return zh.NewProblemDetail(http.StatusServiceUnavailable, "MongoDB not available").Render(w)
		}
		return zh.R.Text(w, http.StatusOK, "ok")
	}

	healthcheck.New(app, healthcheck.Config{
		LivenessHandler:  func(w http.ResponseWriter, r *http.Request) error { return zh.R.Text(w, http.StatusOK, "ok") },
		ReadinessHandler: checkDependencies,
		StartupHandler:   checkDependencies,
	})
}

func setupMiddleware(cfg *config.Config, redisClient *redis.Client, server *zh.Server, tracerImpl *zctracer.OTelTracer) []zh.MiddlewareFunc {
	var middlewares []zh.MiddlewareFunc

	middlewares = append(middlewares,
		compress.New(),
		tracer.New(tracerImpl),
		mediatype.New(mediatype.Config{
			AllowedTypes: []string{
				"application/json",
				"application/vnd.vinylstore.v1+json",
				"application/vnd.vinylstore.v2+json",
			},
			DefaultType:        "application/vnd.vinylstore.v1+json",
			ResponseTypeHeader: "X-Media-Type",
			ResponseTypeFunc:   mediatype.VendorShortType,
		}),
	)

	if cfg.RateLimit.Enabled {
		middlewares = append(middlewares, setupTieredRateLimit(cfg, redisClient, server)...)
	}

	return middlewares
}

// setupAuthenticatedRateLimit creates rate limit middleware for authenticated users
func setupAuthenticatedRateLimit(cfg *config.Config, redisClient *redis.Client, app *zh.Server) zh.MiddlewareFunc {
	if !cfg.RateLimit.Enabled || !cfg.RateLimit.Authenticated.Enabled {
		return nil
	}

	rate := cfg.RateLimit.Authenticated.Rate
	window := cfg.RateLimit.Authenticated.Window
	authStore := zcratelimit.NewRedisStore(redisClient, zcratelimit.RedisStoreConfig{
		Algorithm: ratelimit.TokenBucket,
		Rate:      rate,
		Window:    window,
	})
	authConfig := ratelimit.Config{
		Store:         authStore,
		Rate:          rate,
		Window:        window,
		KeyExtractor:  ratelimit.JWTSubjectKeyExtractor(),
		IncludedPaths: []string{"/orders*", "/auth/logout"},
	}
	app.Logger().Info("Rate limiting: authenticated tier enabled",
		zl.F("limit", fmt.Sprintf("%d/%s", rate, window)),
		zl.F("key_by", "JWT subject"))

	return ratelimit.New(authConfig)
}

// setupTieredRateLimit creates multiple rate limit middlewares for different tiers
func setupTieredRateLimit(cfg *config.Config, redisClient *redis.Client, server *zh.Server) []zh.MiddlewareFunc {
	var middlewares []zh.MiddlewareFunc

	if cfg.RateLimit.Public.Enabled {
		rate := cfg.RateLimit.Authenticated.Rate
		window := cfg.RateLimit.Authenticated.Window
		publicStore := zcratelimit.NewRedisStore(redisClient, zcratelimit.RedisStoreConfig{
			Algorithm: ratelimit.TokenBucket,
			Rate:      rate,
			Window:    window,
		})
		publicConfig := ratelimit.Config{
			Store:         publicStore,
			Rate:          rate,
			Window:        window,
			KeyExtractor:  ratelimit.IPKeyExtractor(),
			IncludedPaths: []string{"/records*", "/inventory", "/users"},
		}
		middlewares = append(middlewares, ratelimit.New(publicConfig))
		server.Logger().Info("Rate limiting: public tier enabled",
			zl.F("limit", fmt.Sprintf("%d/%s", cfg.RateLimit.Public.Rate, window)),
			zl.F("key_by", "IP"))
	}

	if cfg.RateLimit.AuthEndpoints.Enabled {
		rate := cfg.RateLimit.Authenticated.Rate
		window := cfg.RateLimit.Authenticated.Window
		loginStore := zcratelimit.NewRedisStore(redisClient, zcratelimit.RedisStoreConfig{
			Algorithm: ratelimit.TokenBucket,
			Rate:      rate,
			Window:    window,
		})
		loginConfig := ratelimit.Config{
			Store:         loginStore,
			Rate:          rate,
			Window:        window,
			KeyExtractor:  ratelimit.IPKeyExtractor(),
			IncludedPaths: []string{"/auth/login", "/auth/logout"},
		}
		middlewares = append(middlewares, ratelimit.New(loginConfig))
		server.Logger().Info("Rate limiting: auth endpoints tier enabled",
			zl.F("limit", fmt.Sprintf("%d/%s", cfg.RateLimit.AuthEndpoints.Rate, window)),
			zl.F("key_by", "IP"))
	}

	return middlewares
}
