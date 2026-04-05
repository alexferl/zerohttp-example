package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	zh "github.com/alexferl/zerohttp"
	zctracer "github.com/alexferl/zerohttp-contrib/middleware/tracer"
	zl "github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/middleware/securityheaders"
	"github.com/alexferl/zerohttp/middleware/tracer"
	"github.com/alexferl/zerohttp/pagination"
	"github.com/alexferl/zerohttp/pprof"
	"github.com/redis/go-redis/v9"

	"github.com/alexferl/zerohttp-example/config"
	"github.com/alexferl/zerohttp-example/handlers"
	"github.com/alexferl/zerohttp-example/models"
	"github.com/alexferl/zerohttp-example/store"
)

// DataStore extends store.Store with lifecycle methods needed by App
type DataStore interface {
	store.Store
	SeedData()
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
}

const (
	// defaultReadTimeout is the default timeout for reading requests
	defaultReadTimeout = 35 * time.Second
	// defaultPerPage is the default number of items per page for pagination
	defaultPerPage = 50
	// hstsMaxAge is the max age for HSTS header in seconds (1 year)
	hstsMaxAge = 365 * 24 * 60 * 60
)

// App represents the vinyl store application
type App struct {
	cfg         *config.Config
	server      *zh.Server
	dbStore     DataStore
	redisClient *redis.Client
	tracerImpl  *zctracer.OTelTracer
}

// New creates a new App instance
func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	pagination.DefaultPerPage = defaultPerPage

	dbStore, err := store.NewMongoStore(cfg.Mongo.URI, cfg.Mongo.Database)
	if err != nil {
		return nil, err
	}

	tracerImpl, shutdownTracer := setupTracer(ctx, cfg)
	redisClient, redisStorage := setupRedisStorage(ctx, cfg.Redis)
	idempotencyCfg := setupIdempotencyConfig(redisStorage)

	srv := zh.DefaultHTTPServer()
	srv.Addr = cfg.Server.BindAddr
	srv.ReadTimeout = defaultReadTimeout

	appCfg := zh.Config{
		Server: srv,
	}

	if cfg.Server.BindTLSAddr != "" {
		srvTLS := zh.DefaultTLSServer()
		srvTLS.Addr = cfg.Server.BindTLSAddr
		srvTLS.ReadTimeout = defaultReadTimeout

		appCfg.TLS = zh.TLSConfig{
			Server:   srvTLS,
			CertFile: cfg.Server.CertFile,
			KeyFile:  cfg.Server.KeyFile,
		}

		if cfg.Server.EnableHSTS {
			appCfg.SecurityHeaders = securityheaders.Config{
				StrictTransportSecurity: securityheaders.StrictTransportSecurity{
					MaxAge: hstsMaxAge,
				},
			}
		}
	}

	appCfg.Tracer = tracer.Config{TracerField: tracerImpl}
	appCfg.Lifecycle = zh.LifecycleConfig{
		PostStartupHooks: []zh.StartupHookConfig{
			{
				Name: "server-ready",
				Hook: func(ctx context.Context) error {
					zl.GetGlobalLogger().Info("Vinyl Store API is ready")
					return nil
				},
			},
		},
		PostShutdownHooks: []zh.ShutdownHookConfig{
			{
				Name: "tracer-shutdown",
				Hook: func(ctx context.Context) error {
					shutdownTracer()
					return nil
				},
			},
			{
				Name: "mongo-close",
				Hook: func(ctx context.Context) error {
					return dbStore.Close(ctx)
				},
			},
		},
	}

	server := zh.New(appCfg)

	middlewares := setupMiddleware(cfg, redisClient, server, tracerImpl)
	server.Use(middlewares...)
	setupHealthcheck(server, redisClient, dbStore)

	if cfg.App.EnablePprof {
		pprofCfg := pprof.Config{}
		if cfg.App.PprofUsername != "" || cfg.App.PprofPassword != "" {
			pprofCfg.Auth = &pprof.AuthConfig{
				Username: cfg.App.PprofUsername,
				Password: cfg.App.PprofPassword,
			}
		}
		pp := pprof.New(server, pprofCfg)
		if pp.Auth != nil {
			server.Logger().Info("pprof enabled", zl.F("username", pp.Auth.Username), zl.F("password", pp.Auth.Password))
		}
	}

	jwtCfg, err := setupJWTConfig(redisStorage, cfg)
	if err != nil {
		zl.GetGlobalLogger().Fatal("Failed to setup JWT config", zl.E(err))
	}

	h := handlers.New(dbStore, jwtCfg)
	setupRoutes(server, h, jwtCfg, idempotencyCfg, cfg, redisClient)

	return &App{
		cfg:         cfg,
		server:      server,
		dbStore:     dbStore,
		redisClient: redisClient,
		tracerImpl:  tracerImpl,
	}, nil
}

// Start runs the application and blocks until shutdown
func (a *App) Start() error {
	if a.cfg.App.SeedData {
		a.dbStore.SeedData()
	}

	if a.cfg.App.CreateAdmin {
		ctx := context.Background()
		if err := a.createDefaultAdmin(ctx); err != nil {
			return err
		}
	}

	go func() {
		if err := a.server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.server.Logger().Fatal("Server failed to start", zl.E(err))
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	zl.GetGlobalLogger().Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), a.cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		a.server.Logger().Fatal("Server forced to shutdown", zl.E(err))
	}

	a.server.Logger().Info("Server exited gracefully")
	return nil
}

// Server returns the underlying server for testing
func (a *App) Server() *zh.Server {
	return a.server
}

// Store returns the database store for testing
func (a *App) Store() DataStore {
	return a.dbStore
}

// RedisClient returns the redis client for testing
func (a *App) RedisClient() *redis.Client {
	return a.redisClient
}

func (a *App) createDefaultAdmin(ctx context.Context) error {
	_, exists, err := a.dbStore.GetUserByEmail(ctx, a.cfg.App.AdminEmail)
	if err != nil {
		return err
	}
	if exists {
		zl.GetGlobalLogger().Info("Admin user already exists", zl.F("email", a.cfg.App.AdminEmail))
		return nil
	}

	admin, err := models.NewUser(models.UserParams{
		Email:    a.cfg.App.AdminEmail,
		Name:     "Admin User",
		Password: a.cfg.App.AdminPassword,
		Role:     models.RoleAdmin,
	})
	if err != nil {
		return err
	}
	if err := a.dbStore.SaveUser(ctx, admin); err != nil {
		return err
	}
	zl.GetGlobalLogger().Info("Created default admin user", zl.F("email", admin.Email), zl.F("id", admin.ID))
	return nil
}
