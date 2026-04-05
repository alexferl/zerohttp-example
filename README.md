# Vinyl Store API

A production-ready REST API for a vinyl record store built with [zerohttp](https://github.com/alexferl/zerohttp). This example demonstrates clean architecture patterns, JWT authentication, MongoDB persistence, Redis caching, OpenTelemetry tracing and more.

## Features

- **RESTful API** - Clean resource-oriented endpoints for users, records, orders, and inventory
- **JWT Authentication** - ECDSA-signed access tokens with refresh token support
- **Role-Based Access Control** - Admin and user roles with middleware protection
- **MongoDB Persistence** - Production-grade document storage
- **Redis Integration** - Token storage, idempotency keys, and rate limit counters
- **Idempotency Support** - Safe retry handling for POST requests (e.g., order creation)
- **Tiered Rate Limiting** - IP-based for public endpoints, JWT-subject based for authenticated users
- **Request Compression** - Gzip compression for responses
- **OpenTelemetry Tracing** - Distributed tracing with Jaeger
- **Request Validation** - Input validation with detailed error responses
- **Health Checks** - Kubernetes-ready liveness, readiness, and startup probes
- **Structured Logging** - Key=value structured logging with colored output

## Architecture

```
├── app/             # Application bootstrap and server setup
├── cmd/server/      # Application entry point
├── config/          # Configuration management with Viper
├── handlers/        # HTTP handlers using zerohttp
├── models/          # Domain models (User, Record, Order)
├── mocks/           # Generated mocks for testing (mockery)
└── store/           # Data persistence layer
    ├── mongo.go     # MongoDB implementation
    └── store.go     # Store interface definitions
```

## API Endpoints

### Authentication
| Method | Endpoint        | Description               |
|--------|-----------------|---------------------------|
| POST   | `/auth/login`   | Login with email/password |
| POST   | `/auth/refresh` | Refresh access token      |
| POST   | `/auth/logout`  | Logout and revoke tokens  |

### Users
| Method | Endpoint                 | Auth          | Description                          |
|--------|--------------------------|---------------|--------------------------------------|
| POST   | `/users`                 | -             | Create new user                      |
| GET    | `/users`                 | Admin         | List all users                       |
| GET    | `/users/{id}`            | Self or Admin | Get user by ID (own or any if admin) |
| PATCH  | `/users/{id}`            | Self only     | Update own profile                   |
| POST   | `/users/{id}/deactivate` | Self only     | Deactivate own account               |

### Records (Vinyl Inventory)
| Method | Endpoint                | Auth  | Description                 |
|--------|-------------------------|-------|-----------------------------|
| POST   | `/records`              | Admin | Create new record           |
| GET    | `/records`              | -     | List records (with filters) |
| GET    | `/records/{id}`         | -     | Get record by ID            |
| PATCH  | `/records/{id}`         | Admin | Update record               |
| POST   | `/records/{id}/archive` | Admin | Archive record              |

### Orders
| Method | Endpoint              | Auth                 | Description          |
|--------|-----------------------|----------------------|----------------------|
| POST   | `/orders`             | User                 | Create order         |
| GET    | `/orders`             | User                 | List my orders       |
| GET    | `/orders/{id}`        | Order Owner or Admin | Get order details    |
| POST   | `/orders/{id}/cancel` | Order Owner          | Cancel pending order |

### Inventory Management
| Method | Endpoint                  | Auth  | Description           |
|--------|---------------------------|-------|-----------------------|
| GET    | `/inventory`              | Admin | List inventory status |
| POST   | `/inventory/{id}/restock` | Admin | Restock a record      |

### Health & Observability
| Method | Endpoint    | Description                |
|--------|-------------|----------------------------|
| GET    | `/livez`    | Kubernetes liveness probe  |
| GET    | `/readyz`   | Kubernetes readiness probe |
| GET    | `/startupz` | Kubernetes startup probe   |
| GET    | `/metrics`  | Prometheus metrics         |

### Profiling (pprof)

When enabled (`--enable-pprof`), Go runtime profiling endpoints are available at `/debug/pprof/`:

| Endpoint                 | Description                               |
|--------------------------|-------------------------------------------|
| `/debug/pprof/`          | Index page listing all profiles           |
| `/debug/pprof/heap`      | Memory allocation samples                 |
| `/debug/pprof/goroutine` | Stack traces of all goroutines            |
| `/debug/pprof/profile`   | CPU profile (30s sample, use `?seconds=`) |
| `/debug/pprof/trace`     | Execution trace                           |
| `/debug/pprof/block`     | Blocking operations                       |
| `/debug/pprof/mutex`     | Mutex contention                          |

**Security**: pprof endpoints are restricted to localhost by default and require basic authentication. Configure credentials with `--pprof-username` and `--pprof-password`, or leave unset to auto-generate a password (logged on startup).

**Example usage**:
```bash
# Run with custom credentials
go run ./cmd/server --pprof-username=admin --pprof-password=secret

# Get heap profile (with auth)
curl -s -u admin:secret http://localhost:8080/debug/pprof/heap > heap.out

# 30-second CPU profile
curl -s http://localhost:8080/debug/pprof/profile > cpu.out

# View with Go tools
go tool pprof -http=:8081 heap.out
```

## Getting Started

### Prerequisites

- Go 1.26+
- Docker & Docker Compose (for dependencies)
- Make

### Quick Start

1. **Start dependencies:**
   ```bash
   docker-compose up -d
   ```

2. **Run the application:**
   ```bash
   make run
   ```

3. **Create your first user:**
   ```bash
   curl -X POST http://localhost:8080/users \
     -H "Content-Type: application/json" \
     -d '{"email":"user@example.com","name":"User","password":"securepassword123"}'
   ```

### Docker Compose Services

The `docker-compose.yml` includes:
- **MongoDB** (port 27017) - Primary data store
- **Redis** (port 6379) - Token & idempotency storage
- **Jaeger** (port 16686) - Tracing UI

## Configuration

Configuration can be provided via CLI flags, environment variables, or config files. Priority: CLI flags > env vars > config file > defaults.

### Environment Variables

All environment variables are prefixed with `VINYL_`. For example, `--bind-addr` becomes `VINYL_BIND_ADDR`.

### Config Files

The application looks for config files in the following locations:
- `./config.toml` (current directory)
- `/etc/vinylstore/config.toml`
- `$HOME/.config/vinylstore/config.toml`

### Application Settings

| Flag                | Environment            | Default               | Description                       |
|---------------------|------------------------|-----------------------|-----------------------------------|
| `--app-name`        | `VINYL_APP_NAME`       | `vinyl-store-api`     | Application name                  |
| `--seed-data`       | `VINYL_SEED_DATA`      | `true`                | Seed sample records on startup    |
| `--create-admin`    | `VINYL_CREATE_ADMIN`   | `true`                | Create default admin user         |
| `--admin-email`     | `VINYL_ADMIN_EMAIL`    | `admin@example.com`   | Default admin email               |
| `--admin-password`  | `VINYL_ADMIN_PASSWORD` | `admin123`            | Default admin password            |

### Server Settings

| Flag                 | Environment               | Default           | Description                              |
|----------------------|---------------------------|-------------------|------------------------------------------|
| `--bind-addr`        | `VINYL_BIND_ADDR`         | `localhost:8080`  | Server listen address                    |
| `--bind-tls-addr`    | `VINYL_BIND_TLS_ADDR`     | (none)            | Server TLS listen address (optional)     |
| `--shutdown-timeout` | `VINYL_SHUTDOWN_TIMEOUT`  | `30s`             | Graceful shutdown timeout                |
| `--cert-file`        | `VINYL_CERT_FILE`         | (none)            | Path to TLS certificate file             |
| `--key-file`         | `VINYL_KEY_FILE`          | (none)            | Path to TLS key file                     |
| `--enable-hsts`      | `VINYL_ENABLE_HSTS`       | `false`           | Enable HSTS header for HTTPS             |

### MongoDB Settings

| Flag               | Environment              | Default                     | Description             |
|--------------------|--------------------------|-----------------------------|-------------------------|
| `--mongo-uri`      | `VINYL_MONGO_URI`        | `mongodb://localhost:27017` | MongoDB connection URI  |
| `--mongo-database` | `VINYL_MONGO_DATABASE`   | `vinylstore`                | MongoDB database name   |

### Redis Settings

| Flag                 | Environment               | Default          | Description                     |
|----------------------|---------------------------|------------------|---------------------------------|
| `--redis-addr`       | `VINYL_REDIS_ADDR`        | `localhost:6379` | Redis server address            |
| `--redis-key-prefix` | `VINYL_REDIS_KEY_PREFIX`  | `vinylstore:`    | Prefix for all Redis keys       |
| `--redis-lock-ttl`   | `VINYL_REDIS_LOCK_TTL`    | `30s`            | TTL for distributed locks       |

### JWT Authentication Settings

| Flag                     | Environment                  | Default     | Description                        |
|--------------------------|------------------------------|-------------|------------------------------------|
| `--jwt-private-key-path` | `VINYL_JWT_PRIVATE_KEY_PATH` | `jwt.key`   | Path to JWT ECDSA private key file |
| `--jwt-access-ttl`       | `VINYL_JWT_ACCESS_TTL`       | `15m`       | Access token TTL                   |
| `--jwt-refresh-ttl`      | `VINYL_JWT_REFRESH_TTL`      | `168h` (7d) | Refresh token TTL                  |

### OpenTelemetry Tracing Settings

| Flag                | Environment             | Default            | Description                       |
|---------------------|-------------------------|--------------------|-----------------------------------|
| `--tracer-enabled`  | `VINYL_TRACER_ENABLED`  | `true`             | Enable OpenTelemetry tracing      |
| `--tracer-endpoint` | `VINYL_TRACER_ENDPOINT` | `localhost:4317`   | OpenTelemetry collector endpoint  |
| `--tracer-insecure` | `VINYL_TRACER_INSECURE` | `true`             | Use insecure connection           |

### pprof Profiling Settings

| Flag               | Environment             | Default        | Description                       |
|--------------------|-------------------------|----------------|-----------------------------------|
| `--enable-pprof`   | `VINYL_ENABLE_PPROF`    | `false`        | Enable pprof debug endpoints      |
| `--pprof-username` | `VINYL_PPROF_USERNAME`  | `pprof`        | pprof basic auth username         |
| `--pprof-password` | `VINYL_PPROF_PASSWORD`  | (auto-gen)     | pprof basic auth password         |

### Rate Limiting

Tiered rate limiting is enabled by default with three tiers:

| Tier            | Default | Keyed By     | Paths                          |
|-----------------|---------|--------------|--------------------------------|
| Public          | 30/1m   | IP address   | `/records*`, `/inventory`      |
| Auth Endpoints  | 5/1m    | IP address   | `/auth/login`, `/auth/refresh` |
| Authenticated   | 100/1m  | JWT subject  | `/orders*`, `/auth/logout`     |

| Flag                          | Environment                          | Default  | Description                          |
|-------------------------------|--------------------------------------|----------|--------------------------------------|
| `--rate-limit-enabled`        | `VINYL_RATE_LIMIT_ENABLED`           | `true`   | Enable tiered rate limiting          |
| `--rate-limit-public-rate`    | `VINYL_RATE_LIMIT_PUBLIC_RATE`       | `30`     | Public tier: requests per window     |
| `--rate-limit-public-window`  | `VINYL_RATE_LIMIT_PUBLIC_WINDOW`     | `1m`     | Public tier: time window             |
| `--rate-limit-auth-rate`      | `VINYL_RATE_LIMIT_AUTH_RATE`         | `100`    | Authenticated tier: requests/window  |
| `--rate-limit-auth-window`    | `VINYL_RATE_LIMIT_AUTH_WINDOW`       | `1m`     | Authenticated tier: time window      |
| `--rate-limit-login-rate`     | `VINYL_RATE_LIMIT_LOGIN_RATE`        | `5`      | Auth endpoints: requests per window  |
| `--rate-limit-login-window`   | `VINYL_RATE_LIMIT_LOGIN_WINDOW`      | `1m`     | Auth endpoints: time window          |

### Example with custom config:

```bash
# Using CLI flags
go run ./cmd/server --bind-addr 0.0.0.0:9090 --mongo-uri mongodb://mongo:27017 --seed-data=false

# Using environment variables
export VINYL_BIND_ADDR=0.0.0.0:9090
export VINYL_MONGO_URI=mongodb://mongo:27017
export VINYL_SEED_DATA=false
go run ./cmd/server

# Using config file (config.toml)
cat > config.toml << 'EOF'
bind_addr = "0.0.0.0:9090"
mongo_uri = "mongodb://mongo:27017"
seed_data = false
EOF
go run ./cmd/server
```

## Testing

### Unit Tests
Run the test suite with mocks:
```bash
go test ./...
```

### Integration Tests
Run tests against real MongoDB (requires Docker):
```bash
go test ./... -tags=integration
```

### Coverage
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Regenerating Mocks
After modifying store interfaces, regenerate mocks:
```bash
make generate
```

## Project Structure Details

### Store Interface
The application uses a `Store` interface for persistence, making it easy to swap implementations:

```go
type Store interface {
    UserStore
    RecordStore
    OrderStore
}
```

This allows for:
- **MongoDB** - Production implementation (`store/mongo.go`)
- **Mocks** - Generated mocks for unit testing (`mocks/`)

### Handler Pattern
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

### Validation
Input validation uses struct tags with automatic error responses:

```go
type CreateRecordRequest struct {
    Title   string  `json:"title" validate:"required,min=1,max=200"`
    Price   float64 `json:"price" validate:"required,min=0"`
    Genre   string  `json:"genre" validate:"required,oneof=jazz rock electronic ..."`
}
```

## License

MIT License - see [LICENSE](LICENSE) file.
