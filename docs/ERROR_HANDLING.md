# Error Handling Guidelines

This document defines the standard error handling patterns for the atest-ext-ai project.

## Core Principles

Based on [Uber Go Style Guide](https://github.com/uber-go/guide) and [gRPC Go Best Practices](https://github.com/grpc/grpc-go/blob/master/Documentation/rpc-errors.md), our error handling follows these principles:

1. **Handle errors once**: Either handle an error or return it, never both
2. **Preserve error context**: Use `fmt.Errorf` with `%w` to wrap errors
3. **Use appropriate gRPC status codes**: Map internal errors to meaningful gRPC codes
4. **Type-safe error matching**: Use `errors.Is` and `errors.As` for error inspection

## Error Handling Patterns

### ✅ gRPC Service Layer (Recommended)

For all gRPC service methods in `pkg/plugin/service.go`:

```go
// Always return gRPC status errors directly
func (s *AIPluginService) SomeMethod(ctx context.Context, req *Request) (*Response, error) {
    result, err := s.doSomething()
    if err != nil {
        // Return gRPC error with appropriate code
        return nil, status.Errorf(codes.Internal, "failed to do something: %v", err)
    }

    return &Response{Data: result}, nil
}
```

**Never** return success response with error data:

```go
// ❌ BAD: Don't do this
return &server.DataQueryResult{
    Data: []*server.Pair{
        {Key: "error", Value: err.Error()},
        {Key: "success", Value: "false"},
    },
}, nil
```

### ✅ Internal Service Layer

For internal services (AI engine, generators, etc.):

```go
// Wrap errors with context using %w
func (e *Engine) GenerateSQL(ctx context.Context, req *Request) (*Response, error) {
    client, err := e.getClient()
    if err != nil {
        return nil, fmt.Errorf("get AI client: %w", err)
    }

    result, err := client.Generate(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("generate with client: %w", err)
    }

    return result, nil
}
```

### ✅ Error Type Matching

Use defined error types for specific handling:

```go
result, err := doSomething()
if err != nil {
    // Check for specific error types
    if errors.Is(err, ErrProviderNotConfigured) {
        // Handle this specific error gracefully
        logging.Logger.Warn("provider not configured, using default")
        return useDefaultProvider()
    }

    // Return all other errors
    return nil, fmt.Errorf("operation failed: %w", err)
}
```

### ✅ Graceful Degradation

For non-critical operations:

```go
// Log but don't fail
if err := emitMetrics(); err != nil {
    logging.Logger.Warn("failed to emit metrics", zap.Error(err))
    // Continue execution
}
```

## gRPC Status Code Mapping

Map internal errors to appropriate gRPC codes:

| Internal Error | gRPC Code | Usage |
|----------------|-----------|-------|
| `ErrProviderNotConfigured` | `FailedPrecondition` | Service not ready |
| `ErrModelNotFound` | `NotFound` | Resource doesn't exist |
| `ErrInvalidConfig` | `InvalidArgument` | Bad request parameters |
| `ErrConnectionFailed` | `Unavailable` | Service temporarily unavailable |
| Default/Unknown | `Internal` | Unexpected errors |

## Custom Error Types

Define custom errors in `pkg/errors/errors.go`:

```go
var (
    ErrProviderNotConfigured = errors.New("AI provider not configured")
    ErrModelNotFound = errors.New("model not found")
)

// Use ToGRPCError to convert internal errors
grpcErr := errors.ToGRPCError(internalErr)
```

## Anti-Patterns to Avoid

### ❌ Don't log and return

```go
// BAD: Caller might log again, creating noise
if err != nil {
    log.Printf("Error: %v", err)
    return err
}
```

### ❌ Don't lose error context

```go
// BAD: Original error information is lost
if err != nil {
    return fmt.Errorf("operation failed")  // Missing %w
}
```

### ❌ Don't use panic for expected errors

```go
// BAD: Use error returns instead
if config == nil {
    panic("config is nil")
}

// GOOD:
if config == nil {
    return fmt.Errorf("config is required")
}
```

## Testing Error Paths

Always test error handling paths:

```go
func TestHandleError(t *testing.T) {
    _, err := service.Method(ctx, invalidRequest)

    // Verify correct error type
    require.Error(t, err)

    // Verify gRPC status code
    st, ok := status.FromError(err)
    require.True(t, ok)
    assert.Equal(t, codes.InvalidArgument, st.Code())
}
```

## References

- [Uber Go Style Guide - Error Handling](https://github.com/uber-go/guide/blob/master/style.md#error-handling)
- [gRPC Go - RPC Errors](https://github.com/grpc/grpc-go/blob/master/Documentation/rpc-errors.md)
- [Go Blog - Error Handling and Go](https://go.dev/blog/error-handling-and-go)
