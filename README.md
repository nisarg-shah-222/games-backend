# Games App Backend

A clean, modular Go backend boilerplate with a health check API endpoint.

## Features

- 🚀 Clean architecture with separation of concerns
- 📦 Modular structure (handlers, middleware, config, router)
- 🔒 Built-in middleware (logging, recovery, CORS)
- 🐳 Docker support
- ⚙️ Environment-based configuration
- 📝 Well-documented code

## Project Structure

```
backend/
├── main.go                 # Application entry point
├── go.mod                  # Go module definition
├── Dockerfile              # Docker configuration
├── docker-compose.yml      # Docker Compose configuration
├── Makefile                # Build and run commands
├── .env.example            # Environment variables template
└── internal/
    ├── config/             # Configuration management
    │   └── config.go
    ├── handler/            # HTTP handlers
    │   └── health.go
    ├── middleware/         # HTTP middleware
    │   ├── cors.go
    │   ├── logger.go
    │   └── recovery.go
    └── router/             # Route definitions
        └── router.go
```

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Docker (optional, for containerized deployment)

### Installation

1. Clone the repository:
```bash
cd backend
```

2. Install dependencies:
```bash
make deps
# or
go mod download
```

3. Copy environment file:
```bash
cp .env.example .env
```

4. Run the application:
```bash
make run
# or
go run main.go
```

The server will start on `http://localhost:8080`

## API Endpoints

### Health Check

**GET** `/api/v1/health-check`

Returns the health status of the service.

**Response:**
```json
{
  "message": "Service is healthy",
  "status": "ok",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/health-check
```

## Configuration

Environment variables can be set in a `.env` file or as system environment variables:

- `PORT` - Server port (default: 8080)
- `ENVIRONMENT` - Environment mode (default: development)
- `LOG_LEVEL` - Logging level (default: info)
- `API_BASE_URL` - API base path (default: /api/v1)

## Development

### Run locally
```bash
make run
```

### Build
```bash
make build
```

### Run tests
```bash
make test
```

### Format code
```bash
make fmt
```

### Clean build artifacts
```bash
make clean
```

## Docker

### Build Docker image
```bash
make docker-build
```

### Run with Docker Compose
```bash
make docker-run
```

### Stop Docker containers
```bash
make docker-stop
```

## Adding New Endpoints

1. Create a new handler in `internal/handler/`
2. Register the route in `internal/router/router.go`
3. Follow the existing pattern for consistency

Example:
```go
// internal/handler/example.go
func (h *ExampleHandler) GetExample(c *gin.Context) {
    c.JSON(200, gin.H{"message": "example"})
}

// internal/router/router.go
func RegisterExampleRoutes(r *gin.Engine, exampleHandler *handler.ExampleHandler) {
    v1 := r.Group("/api/v1")
    {
        v1.GET("/example", exampleHandler.GetExample)
    }
}
```

## Architecture

The project follows a clean architecture pattern:

- **Handlers**: Handle HTTP requests and responses
- **Middleware**: Cross-cutting concerns (logging, CORS, recovery)
- **Router**: Route definitions and grouping
- **Config**: Configuration management
- **Main**: Application initialization and startup

## License

MIT

