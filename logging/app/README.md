# Go Application with Structured Logging

This is a Go application demonstrating structured logging using Uber's Zap logger.

## Features

- **Structured JSON Logging**: Production-ready JSON formatted logs
- **HTTP Server**: RESTful API with logging middleware
- **Request Context**: Each request gets a unique ID for tracing
- **Error Handling**: Comprehensive error logging with context
- **Background Workers**: Demonstrates logging in background tasks
- **Graceful Shutdown**: Proper shutdown handling
- **Health Checks**: Kubernetes-ready health endpoints

## Endpoints

- `GET /health` - Health check endpoint
- `GET /api/process` - Simulates data processing with structured logging
- `GET /api/error` - Demonstrates error logging

## Log Levels

The application uses the following log levels:
- **Debug**: Detailed information for debugging
- **Info**: General informational messages
- **Warn**: Warning messages
- **Error**: Error messages with full context
- **Fatal**: Critical errors that cause application exit

## Running Locally

```bash
# Install dependencies
go mod download

# Run the application
go run main.go

# Or with development environment
ENVIRONMENT=development go run main.go
```

## Building Docker Image

```bash
docker build -t logging-app:latest .
```

## Deploying to Kubernetes

```bash
# Apply the deployment
kubectl apply -f deployment.yaml

# Check logs
kubectl logs -f -n logging-app -l app=logging-app

# Port forward to test locally
kubectl port-forward -n logging-app svc/logging-app 8080:8080
```

## Testing

```bash
# Health check
curl http://localhost:8080/health

# Process endpoint
curl http://localhost:8080/api/process

# Error endpoint
curl http://localhost:8080/api/error
```

## Log Output Example

### Production Mode (JSON)
```json
{
  "level": "info",
  "timestamp": "2025-12-02T10:15:30.123Z",
  "caller": "main.go:45",
  "msg": "HTTP request completed",
  "method": "GET",
  "path": "/api/process",
  "status_code": 200,
  "duration": "45ms",
  "duration_ms": 45
}
```

### Development Mode (Console)
```
2025-12-02T10:15:30.123+0530    INFO    main.go:45    HTTP request completed    {"method": "GET", "path": "/api/process", "status_code": 200, "duration": "45ms"}
```

## Configuration

Set environment variables:
- `ENVIRONMENT`: Set to `development` for colored console output, or `production` for JSON output

## Structured Logging Best Practices

1. **Always add context**: Use `zap.String()`, `zap.Int()`, etc. to add structured fields
2. **Request IDs**: Include request/correlation IDs for tracing
3. **Error context**: Log errors with all relevant context information
4. **Avoid string formatting**: Don't use `fmt.Sprintf()` in log messages
5. **Use appropriate levels**: Choose the right log level for each message
6. **Child loggers**: Create child loggers with common fields for related operations

## Dependencies

- [Zap](https://github.com/uber-go/zap) - Blazing fast, structured, leveled logging in Go
