# zerohttp-example [![Go Report Card](https://goreportcard.com/badge/github.com/alexferl/zerohttp-example)](https://goreportcard.com/report/github.com/alexferl/zerohttp-example) [![Coverage Status](https://coveralls.io/repos/github/alexferl/zerohttp-example/badge.svg?branch=master)](https://coveralls.io/github/alexferl/zerohttp-example?branch=master)

A production-ready REST API for a vinyl record store built with [zerohttp](https://github.com/alexferl/zerohttp).

## Features

- **RESTful API** - Clean endpoints for users, records, orders, and inventory
- **JWT Authentication** - ECDSA-signed tokens with refresh support
- **RBAC** - Admin and user roles with middleware protection
- **MongoDB & Redis** - Document storage with caching and rate limiting
- **Idempotency** - Safe retry handling for POST requests
- **Rate Limiting** - Tiered limits (IP-based public, JWT-based auth)
- **Observability** - OpenTelemetry tracing, Prometheus metrics, pprof profiling
- **Health Checks** - Kubernetes-ready liveness, readiness, and startup probes

## Quick Start

```bash
# Start dependencies
docker compose up -d

# Run the application
make run

# Create your first user
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","name":"User","password":"securepassword123"}'
```

## API Endpoints

### Authentication
| Method | Endpoint        | Description               |
|--------|-----------------|---------------------------|
| POST   | `/auth/login`   | Login with email/password |
| POST   | `/auth/refresh` | Refresh access token      |
| POST   | `/auth/logout`  | Logout and revoke tokens  |

### Users
| Method | Endpoint                 | Auth       | Description        |
|--------|--------------------------|------------|--------------------|
| POST   | `/users`                 | -          | Create user        |
| GET    | `/users`                 | Admin      | List all users     |
| GET    | `/users/{id}`            | Self/Admin | Get user by ID     |
| PATCH  | `/users/{id}`            | Self       | Update own profile |
| POST   | `/users/{id}/deactivate` | Self       | Deactivate account |

### Records (Vinyl Inventory)
| Method | Endpoint                | Auth  | Description                 |
|--------|-------------------------|-------|-----------------------------|
| POST   | `/records`              | Admin | Create record               |
| GET    | `/records`              | -     | List records (with filters) |
| GET    | `/records/{id}`         | -     | Get record by ID            |
| PATCH  | `/records/{id}`         | Admin | Update record               |
| POST   | `/records/{id}/archive` | Admin | Archive record              |

### Orders
| Method | Endpoint              | Auth        | Description          |
|--------|-----------------------|-------------|----------------------|
| POST   | `/orders`             | User        | Create order         |
| GET    | `/orders`             | User        | List my orders       |
| GET    | `/orders/{id}`        | Owner/Admin | Get order details    |
| POST   | `/orders/{id}/cancel` | Owner       | Cancel pending order |

### Inventory Management
| Method | Endpoint                         | Auth  | Description           |
|--------|----------------------------------|-------|-----------------------|
| GET    | `/inventory`                     | -     | List inventory status |
| POST   | `/inventory/{record_id}/restock` | Admin | Restock a record      |

### Health & Observability
| Endpoint         | Description                    |
|------------------|--------------------------------|
| `/livez`         | Kubernetes liveness probe      |
| `/readyz`        | Kubernetes readiness probe     |
| `/startupz`      | Kubernetes startup probe       |
| `/metrics`       | Prometheus metrics (port 9090) |
| `/debug/pprof/*` | Go profiling (when enabled)    |

## HTTP Headers

| Header            | Description             | Example                              |
|-------------------|-------------------------|--------------------------------------|
| `Accept`          | Response format         | `application/vnd.vinylstore.v1+json` |
| `Accept-Encoding` | Compression (gzip)      | `gzip`                               |
| `Content-Type`    | Request body format     | `application/json`                   |
| `Idempotency-Key` | Idempotency for POST    | `unique-key-123`                     |
| `X-API-Version`   | Response version header | `vinylstore.v1`                      |

## Configuration

Configuration priority: CLI flags > env vars > config file > defaults. All env vars are prefixed with `VINYL_`.

### Common Options

```bash
# CLI flags
go run ./cmd/server --bind-addr 0.0.0.0:9090 --mongo-uri mongodb://mongo:27017

# Environment variables
export VINYL_BIND_ADDR=0.0.0.0:9090
export VINYL_MONGO_URI=mongodb://mongo:27017

# Config file (config.toml)
bind_addr = "0.0.0.0:9090"
mongo_uri = "mongodb://mongo:27017"
```

### Key Settings

| Category   | Key Flags                                            | Defaults                                  |
|------------|------------------------------------------------------|-------------------------------------------|
| Server     | `--bind-addr`, `--shutdown-timeout`                  | `localhost:8080`, `30s`                   |
| MongoDB    | `--mongo-uri`, `--mongo-database`                    | `mongodb://localhost:27017`, `vinylstore` |
| Redis      | `--redis-addr`, `--redis-key-prefix`                 | `localhost:6379`, `vinylstore:`           |
| JWT        | `--jwt-private-key-path`, `--jwt-access-ttl`         | `jwt.key`, `15m`                          |
| Rate Limit | `--rate-limit-public-rate`, `--rate-limit-auth-rate` | `30/m`, `100/m`                           |
| Tracing    | `--tracer-enabled`, `--tracer-endpoint`              | `true`, `localhost:4317`                  |

## Rate Limiting

Three tiers by default:

| Tier           | Limit   | Keyed By    | Paths                               |
|----------------|---------|-------------|-------------------------------------|
| Public         | 30/min  | IP          | `/records*`, `/inventory`, `/users` |
| Auth Endpoints | 5/min   | IP          | `/auth/login`, `/auth/refresh`      |
| Authenticated  | 100/min | JWT subject | `/orders*`, `/auth/logout`          |

## Testing

```bash
# Unit tests
go test ./...

# Integration tests (requires Docker)
go test ./... -tags=integration

# Coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Regenerate mocks
make generate
```

## Architecture

```
├── app/          # Application bootstrap and server setup
├── cmd/server/   # Application entry point
├── config/       # Configuration management with Viper
├── handlers/     # HTTP handlers using zerohttp
├── models/       # Domain models (User, Record, Order)
├── mocks/        # Generated mocks for testing
└── store/        # Data persistence layer (MongoDB interface)
```

Handlers use zerohttp's error-returning pattern:

```go
func (h *Handler) GetRecord(w http.ResponseWriter, r *http.Request) error {
    id := r.PathValue("id")
    record, ok := h.store.GetRecord(r.Context(), id)
    if !ok {
        return zh.NotFoundError("record not found")
    }
    return zh.JSON(w, http.StatusOK, record)
}
```

## License

MIT License - see [LICENSE](LICENSE) file.
