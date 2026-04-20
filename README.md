# Rate-Limited API Service

A high-performance, production-ready backend service demonstrating rate limiting, concurrency handling, and modern Go development practices using the Gin framework and Redis.

## Features

- **Rate Limiting**: 5 requests per user per minute with Redis-backed storage
- **Concurrent Processing**: Thread-safe implementation with local caching for performance
- **Production Ready**: Comprehensive error handling, security headers, and graceful shutdown
- **Statistics**: Per-user request statistics tracking
- **Testing**: Full test suite with concurrent request validation

## API Endpoints

### POST /request
Processes a request for a user with rate limiting.

**Request Body:**
```json
{
  "user_id": "user123",
  "payload": "Your request data"
}
```

**Success Response (200):**
```json
{
  "message": "Request processed successfully",
  "user_id": "user123",
  "remaining_requests": 4
}
```

**Rate Limited Response (429):**
```json
{
  "error": "Rate limit exceeded",
  "limit": 5,
  "current": 5,
  "reset_time": 1713638400,
  "retry_after": 45
}
```

### GET /stats
Returns per-user request statistics.

**Query Parameters:**
- `user_id` (required): The user ID to get stats for

**Response:**
```json
{
  "user_id": "user123",
  "total_requests": 25,
  "current_requests": 3,
  "reset_time": 1713638400
}
```

## Sample API Calls

### Make a Request
```bash
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "payload": "Sample request data"
  }'
```

**Success Response:**
```json
{
  "message": "Request processed successfully",
  "user_id": "user123",
  "remaining_requests": 4
}
```

**Rate Limited Response:**
```json
{
  "error": "Rate limit exceeded",
  "limit": 5,
  "current": 5,
  "reset_time": 1713638400,
  "retry_after": 45
}
```

### Get User Statistics
```bash
curl "http://localhost:8080/stats?user_id=user123"
```

**Response:**
```json
{
  "user_id": "user123",
  "total_requests": 3,
  "current_requests": 3,
  "reset_time": 1713638400
}
```

## Setup & Installation

### Prerequisites

- Go 1.21 or higher
- Redis server running on localhost:6379 (or configurable via environment)

### Quick Start

1. **Clone and setup:**
```bash
git clone <repository-url>
cd rate-limited-api
go mod tidy
```

2. **Start Redis:**
```bash
# Install Redis locally (macOS with Homebrew)
brew install redis
brew services start redis

# Or download and run Redis manually
# Download from https://redis.io/download
redis-server
```

3. **Run the service:**
```bash
# Run from project root
go run cmd/api/main.go

# Or build and run
go build -o bin/api cmd/api/main.go
./bin/api
```

The service will start on `http://localhost:8080`

### Configuration

Environment variables:

|     Variable    |      Default     |       Description       |
|-----------------|------------------|-------------------------|
| `PORT`          | `8080`           | Server port             |
| `REDIS_ADDR`    | `localhost:6379` | Redis server address    |
| `QUEUE_WORKERS` | `10`             | Number of queue workers |

Example with custom configuration:
```bash
PORT=3000 REDIS_ADDR=redis.example.com:6379 QUEUE_WORKERS=20 go run cmd/api/main.go
```

### Project Structure

```
rate-limited-api/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── handlers/                # HTTP request handlers
│   │   └── handlers.go
│   ├── middleware/              # HTTP middleware
│   │   └── middleware.go
│   └── services/               # Business logic services
│       └── services.go
├── pkg/                        # Reusable packages
│   ├── config/                 # Configuration management
│   │   └── config.go
│   ├── queue/                  # Request queue implementation
│   │   └── queue.go
│   └── ratelimiter/            # Rate limiting logic
│       └── ratelimiter.go
├── main_test.go                # Integration tests
├── go.mod                      # Go module definition
└── README.md                   # Documentation
```

### Build and Run

```bash
# Development mode with hot reload
go run cmd/api/main.go

# Production build
go build -o bin/api cmd/api/main.go
./bin/api

# Run with custom configuration
PORT=3000 REDIS_ADDR=localhost:6380 go run cmd/api/main.go
```

## Testing

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -v ./... -cover

# Run specific test
go test -v ./... -run TestRequestEndpoint_Success

# Run tests for specific package
go test -v cmd/api -run TestRequestEndpoint_Success
```

### Test Coverage

The test suite includes:
- Unit tests for all endpoints
- Concurrent request handling validation
- Rate limiting behavior verification
- Error condition testing
- Edge case validation

## Architecture & Design Decisions

### Rate Limiting Strategy

**Redis-based with Local Caching:**
- Primary rate limiting stored in Redis for distributed consistency
- Local in-memory cache for frequently accessed users to reduce Redis load
- Atomic operations using Redis pipelines for thread safety
- TTL-based expiration for automatic cleanup

**Why this approach:**
- **Scalability**: Redis allows horizontal scaling across multiple instances
- **Performance**: Local cache reduces Redis round-trips for hot users
- **Consistency**: Redis ensures all instances see the same rate limits
- **Reliability**: TTL prevents memory leaks and handles cleanup automatically

### Concurrency Handling

**Multi-layered Approach:**
1. **Gin's built-in concurrency**: Handles HTTP request concurrency
2. **Redis atomic operations**: Ensures rate limiting accuracy under load
3. **Local cache with mutexes**: Provides thread-safe in-memory operations
4. **Worker pool pattern**: Queue system with configurable workers

**Thread Safety Guarantees:**
- Redis operations are atomic
- Local cache uses sync.RWMutex for safe concurrent access
- Queue workers are isolated goroutines with proper synchronization

### Queue System Design

**Worker Pool Pattern:**
- Configurable number of workers (default: 10)
- Bounded queue (1000 items) to prevent memory exhaustion
- Exponential backoff retry logic
- Graceful degradation when queue is full

**Benefits:**
- **Load Management**: Smooths traffic spikes
- **Retry Logic**: Handles transient failures automatically
- **Resource Control**: Bounded queue prevents OOM
- **Monitoring**: Queue size visibility for scaling decisions

### Error Handling Strategy

**Layered Error Handling:**
1. **Input Validation**: JSON schema validation with clear error messages
2. **Business Logic**: Meaningful error responses with appropriate HTTP codes
3. **Infrastructure**: Graceful degradation when Redis is unavailable
4. **Security**: Sanitized error messages to prevent information leakage

## Performance Considerations

### Optimizations

- **Connection Pooling**: Redis client with optimized pool settings
- **Pipeline Operations**: Batch Redis commands to reduce network latency
- **Local Caching**: Reduces Redis calls for frequently accessed users
- **Timeout Configuration**: Appropriate timeouts for all operations
- **Memory Management**: Bounded queues and TTL-based cleanup

### Scalability

- **Horizontal Scaling**: Stateless design allows multiple instances
- **Redis Cluster**: Can be upgraded to Redis Cluster for high availability
- **Load Balancing**: Compatible with any HTTP load balancer
- **Monitoring**: Built-in metrics for queue status and health

## Future Improvements

1. **Dynamic Rate Limiting**: Per-user or tier-based rate limits
2. **Redis Cluster Support**: High availability and horizontal scaling
3. **Advanced Monitoring**: Prometheus metrics and Grafana dashboards
4. **Circuit Breaker**: Handle Redis failures gracefully
5. **Distributed Tracing**: OpenTelemetry integration
6. **API Versioning**: Support for multiple API versions
7. **Authentication**: JWT or OAuth2 integration
8. **Rate Limit Algorithms**: Support for sliding window, token bucket, etc.