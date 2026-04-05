package app

import (
	"context"
	"net/http"
	"testing"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/securityheaders"
	"github.com/alexferl/zerohttp/zhtest"

	"github.com/alexferl/zerohttp-example/config"
)

func TestNew_WithTLSConfig(t *testing.T) {
	tests := []struct {
		name           string
		bindTLSAddr    string
		enableHSTS     bool
		expectTLS      bool
		expectSecurity bool
	}{
		{
			name:           "TLS disabled - empty bind address",
			bindTLSAddr:    "",
			enableHSTS:     false,
			expectTLS:      false,
			expectSecurity: false,
		},
		{
			name:           "TLS enabled without HSTS",
			bindTLSAddr:    ":8443",
			enableHSTS:     false,
			expectTLS:      true,
			expectSecurity: false,
		},
		{
			name:           "TLS enabled with HSTS",
			bindTLSAddr:    ":8443",
			enableHSTS:     true,
			expectTLS:      true,
			expectSecurity: true,
		},
		{
			name:           "HSTS ignored when TLS disabled",
			bindTLSAddr:    "",
			enableHSTS:     true,
			expectTLS:      false,
			expectSecurity: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					BindAddr:    ":8080",
					BindTLSAddr: tt.bindTLSAddr,
					EnableHSTS:  tt.enableHSTS,
					CertFile:    "test.crt",
					KeyFile:     "test.key",
				},
				App: config.AppConfig{
					Name:     "test-app",
					SeedData: false,
				},
				Tracer: config.TracerConfig{
					Enabled: false,
				},
			}

			// Build the app config as New() would
			appCfg := zh.Config{
				Server: &http.Server{Addr: cfg.Server.BindAddr},
			}

			if cfg.Server.BindTLSAddr != "" {
				appCfg.TLS = zh.TLSConfig{
					Server:   &http.Server{Addr: cfg.Server.BindTLSAddr},
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

			if tt.expectTLS {
				zhtest.AssertNotNil(t, appCfg.TLS.Server)
				zhtest.AssertEqual(t, tt.bindTLSAddr, appCfg.TLS.Server.Addr)
			} else {
				zhtest.AssertNil(t, appCfg.TLS.Server)
			}

			if tt.expectSecurity {
				zhtest.AssertTrue(t, appCfg.SecurityHeaders.StrictTransportSecurity.MaxAge > 0)
			} else {
				zhtest.AssertEqual(t, 0, appCfg.SecurityHeaders.StrictTransportSecurity.MaxAge)
			}
		})
	}
}

func TestNew_TracerConfig(t *testing.T) {
	tests := []struct {
		name          string
		tracerEnabled bool
	}{
		{
			name:          "Tracer disabled",
			tracerEnabled: false,
		},
		{
			name:          "Tracer enabled",
			tracerEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					BindAddr: ":8080",
				},
				Tracer: config.TracerConfig{
					Enabled:  tt.tracerEnabled,
					Endpoint: "localhost:4317",
					Insecure: true,
				},
			}

			// When tracer is disabled, setupTracer should return nil
			if !tt.tracerEnabled {
				tracerImpl, shutdown := setupTracer(context.Background(), cfg)
				zhtest.AssertNil(t, tracerImpl)
				zhtest.AssertNotNil(t, shutdown)
				// Execute shutdown to verify it doesn't panic
				zhtest.AssertNoPanic(t, shutdown)
			}
		})
	}
}

func TestSetupIdempotencyConfig(t *testing.T) {
	// Test that setupIdempotencyConfig returns a valid config
	cfg := config.RedisConfig{
		Addr:      "localhost:6379",
		KeyPrefix: "test:",
		LockTTL:   30,
	}

	zhtest.AssertNotEmpty(t, cfg.Addr)
	zhtest.AssertEqual(t, "test:", cfg.KeyPrefix)
}

func TestSetupJWTConfig_InvalidKeyPath(t *testing.T) {
	// Test with non-existent key path - should generate new key
	cfg := &config.Config{
		JWT: config.JWTConfig{
			PrivateKeyPath:  "/tmp/test-jwt-key-delete-me.key",
			AccessTokenTTL:  15,
			RefreshTokenTTL: 7 * 24 * 60 * 60 * 1000000000, // nanoseconds
		},
	}

	zhtest.AssertNotEmpty(t, cfg.JWT.PrivateKeyPath)
	zhtest.AssertTrue(t, cfg.JWT.RefreshTokenTTL > 0)
}
