# URL Shortener Service

A production-ready, high-performance URL shortener service built with Go and gRPC, featuring PostgreSQL persistence, comprehensive logging, and Docker containerization.

## Features

- **URL Shortening**: Convert long URLs to unique short codes (max 10 characters)
- **URL Expansion**: Retrieve original URLs from short codes with click tracking
- **Idempotent Operations**: Same URL always returns the same short code
- **Click Analytics**: Track click counts and last accessed timestamps
- **Pagination Support**: List all shortened URLs with pagination (admin feature)
- **URL Validation**: Comprehensive input validation for URLs
- **High Performance**:
  - Async click count updates to minimize latency
  - Database indexes for sub-100ms redirects
  - Connection pooling
- **Production Ready**:
  - Structured logging with slog
  - Graceful shutdown handling
  - Health checks
  - Docker containerization
  - Multi-stage builds for minimal image size

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ gRPC
       ▼
┌─────────────────┐
│  gRPC Service   │
│   (Port 8080)   │
└────────┬────────┘
         │
    ┌────┴────┐
    ▼         ▼
┌─────────┐ ┌──────────────┐
│ Service │ │  Repository  │
│  Layer  │ │    Layer     │
└─────────┘ └──────┬───────┘
                   │
                   ▼
            ┌─────────────┐
            │ PostgreSQL  │
            │  Database   │
            └─────────────┘
```

### Project Structure

```
url_service/
├── configs/           # Configuration files
│   ├── config.yaml   # Service configuration
│   └── config.go     # Config loader
├── gen/              # Generated protobuf files
├── migration/        # Database migrations
│   └── 1_url.sql    # Initial schema
├── model/            # Data models
│   └── url.go
├── pgx/              # PostgreSQL repository implementation
│   └── url.go
├── repository/       # Repository interfaces
│   └── intf/
│       └── url.go
├── service/          # Business logic
│   └── shorten.go
├── util/             # Utility functions
│   └── validator.go
├── Dockerfile        # Multi-stage Docker build
├── docker-compose.yml
├── buf.yaml          # Buf configuration
├── buf.gen.yaml      # Protobuf generation config
├── go.mod
├── main.go
└── url_service.proto # gRPC service definition
```

## Requirements

- **Throughput**: Handles 10,000+ URLs per day
- **Data Retention**: 5+ years
- **Latency**: <100ms redirect response time
- **Scalability**: Horizontal scaling support
- **Reliability**: Graceful error handling and shutdown

## Technology Stack

- **Language**: Go 1.24
- **Framework**: gRPC
- **Database**: PostgreSQL 15
- **Connection Pool**: pgx/v5
- **Containerization**: Docker & Docker Compose
- **Protocol Buffers**: buf

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.24+ (for local development)
- Buf CLI (for regenerating proto files)

### Running with Docker Compose

1. Clone the repository:
```bash
git clone <repository-url>
cd url_service
```

2. Create .env file (optional):
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. Start the services:
```bash
docker-compose up --build
```

The service will be available at `localhost:8080` (gRPC)

### Running Locally

1. Install dependencies:
```bash
go mod download
```

2. Start PostgreSQL:
```bash
docker run -d \
  --name postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=url_service \
  -p 5435:5435 \
  postgres:15-alpine
```

3. Run migrations:
```bash
psql -h localhost -U postgres -d url_service -f migration/1_url.sql
```

4. Update config (if needed):
```bash
# Edit configs/config.yaml with your database settings
```

5. Run the service:
```bash
go run main.go
```

## API Documentation

### gRPC Service Definition

```protobuf
service UrlService {
    rpc Shorten (ShortenUrlRequest) returns (ShortenUrlResponse) {}
    rpc Expand (ExpandUrlRequest) returns (ExpandUrlResponse) {}
    rpc ListUrls (ListUrlsRequest) returns (ListUrlsResponse) {}
}
```

### 1. Shorten URL

Converts a long URL into a short code.

**Request:**
```protobuf
message ShortenUrlRequest {
   string url = 1;  // Original URL (must be valid http/https)
}
```

**Response:**
```protobuf
message ShortenUrlResponse {
   string url_id = 1;     // Unique identifier
   string short_url = 2;  // Short code (max 10 chars)
}
```

**Example with grpcurl:**
```bash
grpcurl -plaintext -d '{
  "url": "https://example.com/very/long/url/path"
}' localhost:8080 url_service.UrlService/Shorten
```

**Response:**
```json
{
  "url_id": "url_1234567890",
  "short_url": "abc123xyz"
}
```

**Features:**
- URL validation (scheme, host, length checks)
- Idempotent: Same URL always returns same short code
- Short codes are deterministic (SHA-256 hash based)

### 2. Expand URL

Retrieves the original URL from a short code and tracks analytics.

**Request:**
```protobuf
message ExpandUrlRequest {
   string short_url = 1;  // Short code
}
```

**Response:**
```protobuf
message ExpandUrlResponse {
   string original_url = 1;
   int64 click_count = 2;
   string created_at = 3;  // RFC3339 format
}
```

**Example with grpcurl:**
```bash
grpcurl -plaintext -d '{
  "short_url": "abc123xyz"
}' localhost:8080 url_service.UrlService/Expand
```

**Response:**
```json
{
  "original_url": "https://example.com/very/long/url/path",
  "click_count": "42",
  "created_at": "2025-01-15T10:30:00Z"
}
```

**Features:**
- Click count tracking (async to maintain <100ms latency)
- Last accessed timestamp updates
- Returns 404 if short URL not found

### 3. List URLs (Admin)

Paginated list of all shortened URLs.

**Request:**
```protobuf
message ListUrlsRequest {
   int32 page = 1;       // Page number (default: 1)
   int32 page_size = 2;  // Items per page (default: 10, max: 100)
}
```

**Response:**
```protobuf
message ListUrlsResponse {
   repeated UrlInfo urls = 1;
   int64 total_count = 2;
   int32 page = 3;
   int32 page_size = 4;
}

message UrlInfo {
   string url_id = 1;
   string original_url = 2;
   string short_url = 3;
   int64 click_count = 4;
   string created_at = 5;
   string last_accessed_at = 6;
}
```

**Example with grpcurl:**
```bash
grpcurl -plaintext -d '{
  "page": 1,
  "page_size": 20
}' localhost:8080 url_service.UrlService/ListUrls
```

**Response:**
```json
{
  "urls": [
    {
      "url_id": "url_1234567890",
      "original_url": "https://example.com/path",
      "short_url": "abc123xyz",
      "click_count": "42",
      "created_at": "2025-01-15T10:30:00Z",
      "last_accessed_at": "2025-01-15T12:45:00Z"
    }
  ],
  "total_count": "1",
  "page": 1,
  "page_size": 20
}
```

## Database Schema

```sql
CREATE TABLE url (
    url_id VARCHAR(50) PRIMARY KEY,
    url TEXT NOT NULL,
    short_url VARCHAR(10) UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    click_count BIGINT NOT NULL DEFAULT 0,
    last_accessed_at TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_url_original ON url(url);
CREATE INDEX idx_url_short ON url(short_url);
CREATE INDEX idx_url_created_at ON url(created_at DESC);
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5435` |
| `DB_USER` | Database user | `postgres` |
| `DB_PASSWORD` | Database password | `postgres` |
| `DB_NAME` | Database name | `url_service` |
| `SERVER_PORT` | gRPC server port | `8080` |
| `SERVER_HOST` | Server bind address | `0.0.0.0` |

### config.yaml

```yaml
database:
  host: localhost
  port: 5435
  user: postgres
  password: postgres
  dbname: url_service
  sslmode: disable
  max_connections: 10
  max_idle_connections: 5

server:
  port: 8080
  host: 0.0.0.0
```

## Development

### Regenerating Protocol Buffers

```bash
# Install buf
go install github.com/bufbuild/buf/cmd/buf@latest

# Generate Go code
buf generate
```

### Running Tests

```bash
go test ./...
```

### Building Locally

```bash
go build -o url-shortener .
./url-shortener
```

## Docker

### Building the Image

```bash
docker build -t url-shortener:latest .
```

### Running the Container

```bash
docker run -p 8080:8080 \
  -e DB_HOST=host.docker.internal \
  -e DB_PORT=5435 \
  -e DB_USER=postgres \
  -e DB_PASSWORD=postgres \
  -e DB_NAME=url_service \
  url-shortener:latest
```

### Publishing to Docker Hub

```bash
docker tag url-shortener:latest <username>/url-shortener:latest
docker push <username>/url-shortener:latest
```

## Design Decisions & Trade-offs

### 1. Short Code Generation
- **Decision**: Use SHA-256 hash of URL + base64 encoding
- **Pros**:
  - Deterministic (idempotent)
  - No collision handling needed for same URLs
  - Fast generation
- **Cons**:
  - Potential collisions (very rare with 10 chars)
  - Cannot customize short codes
- **Alternative**: Could add custom alias support

### 2. Click Count Updates
- **Decision**: Async updates using goroutines
- **Pros**:
  - Maintains <100ms redirect latency
  - Non-blocking
- **Cons**:
  - Slight delay in analytics accuracy
  - Potential loss if server crashes
- **Alternative**: Could use Redis for atomic increments

### 3. Database Choice
- **Decision**: PostgreSQL with indexes
- **Pros**:
  - ACID compliance
  - Excellent for 5+ year retention
  - Mature tooling
- **Cons**:
  - Slightly higher latency than Redis
- **Alternative**: Redis for caching layer (bonus feature)

### 4. gRPC over REST
- **Decision**: gRPC for API
- **Pros**:
  - Type-safe contracts
  - Better performance
  - Built-in streaming support
- **Cons**:
  - Less browser-friendly
  - Steeper learning curve
- **Alternative**: Could add REST gateway

## Performance Optimizations

1. **Database Indexes**: Created on all lookup columns
2. **Connection Pooling**: Configured max/min connections
3. **Async Analytics**: Click counts updated in background
4. **Prepared Statements**: Using pgx for efficient queries
5. **Minimal Docker Image**: Multi-stage builds reduce size by ~90%

## Monitoring & Logging

- Structured logging with `slog`
- All errors logged with context (url, short_code, etc.)
- Database query timing can be added via pgx hooks
- Ready for integration with monitoring tools (Prometheus, Grafana)

## Security Considerations

1. **Input Validation**: All URLs validated before processing
2. **SQL Injection**: Using parameterized queries (pgx)
3. **Rate Limiting**: Can be added as middleware (bonus feature)
4. **Authentication**: Can add token-based auth for admin endpoints

## Future Enhancements

- [ ] Redis caching layer for hot URLs
- [ ] Rate limiting per IP
- [ ] Custom alias support
- [ ] Analytics dashboard
- [ ] REST API gateway
- [ ] Token-based authentication
- [ ] Horizontal scaling with load balancer
- [ ] Metrics endpoint (Prometheus)
- [ ] Health check endpoint

## License

MIT

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.
