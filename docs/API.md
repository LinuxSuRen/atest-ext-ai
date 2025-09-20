# atest-ext-ai API Documentation

## Overview

The atest-ext-ai plugin provides AI-powered SQL generation capabilities through a gRPC interface that integrates with the main API Testing Tool. This document details all available endpoints, request/response formats, and usage examples.

## Table of Contents

- [Architecture](#architecture)
- [gRPC Interface](#grpc-interface)
- [HTTP API Endpoints](#http-api-endpoints)
- [Request/Response Examples](#requestresponse-examples)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [Authentication](#authentication)
- [SDKs and Client Libraries](#sdks-and-client-libraries)

## Architecture

The plugin implements a gRPC service that communicates via Unix socket with the main API Testing Tool:

```
Client Request → Main API Testing Tool → gRPC Bridge → Unix Socket → atest-ext-ai Plugin → AI Provider
```

## gRPC Interface

### Service Definition

The plugin implements the `Loader` service from the main project's protobuf definition:

```protobuf
service Loader {
  rpc Load(LoadRequest) returns (LoadResponse);
  rpc GetPairs(PairsRequest) returns (PairsResponse);
}
```

### LoadRequest

Handles AI-powered SQL generation requests.

```protobuf
message LoadRequest {
  string name = 1;           // Store name (should be "ai")
  repeated Pair properties = 2; // Configuration properties
  string query = 3;          // Natural language query
}

message Pair {
  string key = 1;
  string value = 2;
}
```

#### Required Properties

| Property | Description | Example |
|----------|-------------|---------|
| `ai_provider` | AI provider to use | `local`, `openai`, `claude` |
| `model` | AI model name | `codellama`, `gpt-4`, `claude-3` |
| `database_type` | Target database type | `mysql`, `postgresql`, `sqlite` |

#### Optional Properties

| Property | Description | Default |
|----------|-------------|---------|
| `confidence_threshold` | Minimum confidence score | `0.7` |
| `enable_sql_execution` | Allow SQL execution | `false` |
| `ollama_endpoint` | Ollama endpoint URL | `http://localhost:11434` |
| `max_tokens` | Maximum tokens for AI response | `1000` |
| `temperature` | AI model temperature | `0.3` |

### LoadResponse

Returns generated SQL and metadata.

```protobuf
message LoadResponse {
  repeated Pair data = 1;
  AIInfo ai_info = 2;
}

message AIInfo {
  int32 processing_time_ms = 1;
  string model_used = 2;
  float confidence_score = 3;
  string provider = 4;
}
```

#### Response Data Pairs

| Key | Description | Type |
|-----|-------------|------|
| `generated_sql` | Generated SQL query | `string` |
| `explanation` | Human-readable explanation | `string` |
| `confidence_score` | AI confidence level | `float` |
| `database_type` | Target database type | `string` |
| `estimated_complexity` | Query complexity rating | `string` |
| `suggested_indexes` | Recommended indexes | `string` |
| `execution_plan` | Query execution hints | `string` |

### GetPairsRequest

Retrieves available configuration options.

```protobuf
message PairsRequest {
  string name = 1;
}
```

### GetPairsResponse

Returns configuration schema and defaults.

```protobuf
message PairsResponse {
  repeated Pair data = 1;
}
```

## HTTP API Endpoints

When integrated with the main API Testing Tool, the following HTTP endpoints become available:

### Generate SQL Query

**POST** `/api/v1/data/query`

Generate SQL from natural language description.

#### Request Body

```json
{
  "type": "ai",
  "natural_language": "Find all active users who registered last month",
  "database_type": "mysql",
  "options": {
    "ai_provider": "local",
    "model": "codellama",
    "confidence_threshold": 0.8
  }
}
```

#### Response

```json
{
  "data": [
    {
      "key": "generated_sql",
      "value": "SELECT u.id, u.username, u.email, u.created_at FROM users u WHERE u.status = 'active' AND u.created_at >= DATE_SUB(NOW(), INTERVAL 1 MONTH) ORDER BY u.created_at DESC"
    },
    {
      "key": "explanation",
      "value": "This query selects all user information for users with active status who were created within the last month, ordered by creation date (newest first)"
    },
    {
      "key": "confidence_score",
      "value": "0.92"
    },
    {
      "key": "estimated_complexity",
      "value": "medium"
    },
    {
      "key": "suggested_indexes",
      "value": "INDEX idx_users_status_created (status, created_at)"
    }
  ],
  "ai_info": {
    "processing_time_ms": 1247,
    "model_used": "codellama",
    "confidence_score": 0.92,
    "provider": "local"
  }
}
```

### Execute SQL Query

**POST** `/api/v1/data/execute`

Execute generated SQL query (when enabled).

#### Request Body

```json
{
  "type": "ai",
  "sql": "SELECT * FROM users WHERE status = 'active' LIMIT 10",
  "database_config": {
    "type": "mysql",
    "host": "localhost",
    "port": 3306,
    "database": "testdb",
    "username": "testuser",
    "password": "testpass"
  }
}
```

#### Response

```json
{
  "data": [
    {
      "key": "results",
      "value": "[{\"id\":1,\"username\":\"john_doe\",\"email\":\"john@example.com\",\"status\":\"active\"}]"
    },
    {
      "key": "rows_affected",
      "value": "1"
    },
    {
      "key": "execution_time_ms",
      "value": "23"
    }
  ],
  "ai_info": {
    "processing_time_ms": 45,
    "model_used": "codellama",
    "confidence_score": 0.95,
    "provider": "local"
  }
}
```

### Get Plugin Status

**GET** `/api/v1/plugins/ai/status`

Retrieve plugin health and status information.

#### Response

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "ai_providers": {
    "local": {
      "status": "available",
      "endpoint": "http://localhost:11434",
      "models": ["codellama", "mistral", "llama2"]
    },
    "openai": {
      "status": "configured",
      "models": ["gpt-4", "gpt-3.5-turbo"]
    }
  },
  "supported_databases": ["mysql", "postgresql", "sqlite"],
  "uptime_seconds": 3600,
  "requests_processed": 127,
  "average_response_time_ms": 1834
}
```

### List Available Models

**GET** `/api/v1/ai/models`

List available AI models for each provider.

#### Query Parameters

- `provider` (optional): Filter by specific provider (`local`, `openai`, `claude`)

#### Response

```json
{
  "models": [
    {
      "provider": "local",
      "name": "codellama",
      "description": "Code generation and completion model",
      "parameter_count": "7B",
      "context_length": 4096,
      "status": "available"
    },
    {
      "provider": "openai",
      "name": "gpt-4",
      "description": "Most capable GPT-4 model",
      "context_length": 8192,
      "status": "available"
    }
  ]
}
```

## Request/Response Examples

### Example 1: Simple Query Generation

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/data/query \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ai",
    "natural_language": "Get all products with price above 100",
    "database_type": "postgresql"
  }'
```

**Response:**
```json
{
  "data": [
    {
      "key": "generated_sql",
      "value": "SELECT * FROM products WHERE price > 100 ORDER BY price DESC"
    },
    {
      "key": "explanation",
      "value": "Retrieves all product records where the price field exceeds 100, sorted by price in descending order"
    },
    {
      "key": "confidence_score",
      "value": "0.95"
    }
  ],
  "ai_info": {
    "processing_time_ms": 892,
    "model_used": "codellama",
    "confidence_score": 0.95,
    "provider": "local"
  }
}
```

### Example 2: Complex Join Query

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/data/query \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ai",
    "natural_language": "Find customers who made orders in the last 30 days with their total order value",
    "database_type": "mysql",
    "options": {
      "ai_provider": "openai",
      "model": "gpt-4",
      "confidence_threshold": 0.85
    }
  }'
```

**Response:**
```json
{
  "data": [
    {
      "key": "generated_sql",
      "value": "SELECT c.id, c.name, c.email, COUNT(o.id) as order_count, SUM(o.total_amount) as total_value FROM customers c INNER JOIN orders o ON c.id = o.customer_id WHERE o.created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY) GROUP BY c.id, c.name, c.email ORDER BY total_value DESC"
    },
    {
      "key": "explanation",
      "value": "This query joins customers and orders tables to find customers who placed orders in the last 30 days, calculating their total order count and value, ordered by total value"
    },
    {
      "key": "confidence_score",
      "value": "0.89"
    },
    {
      "key": "estimated_complexity",
      "value": "high"
    },
    {
      "key": "suggested_indexes",
      "value": "INDEX idx_orders_customer_created (customer_id, created_at), INDEX idx_orders_created (created_at)"
    }
  ],
  "ai_info": {
    "processing_time_ms": 2134,
    "model_used": "gpt-4",
    "confidence_score": 0.89,
    "provider": "openai"
  }
}
```

### Example 3: Schema-aware Query

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/data/query \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ai",
    "natural_language": "Show monthly sales trends for the current year",
    "database_type": "postgresql",
    "schema_context": {
      "tables": [
        {
          "name": "sales",
          "columns": ["id", "amount", "sale_date", "product_id", "customer_id"]
        },
        {
          "name": "products",
          "columns": ["id", "name", "category", "price"]
        }
      ]
    }
  }'
```

**Response:**
```json
{
  "data": [
    {
      "key": "generated_sql",
      "value": "SELECT DATE_TRUNC('month', s.sale_date) as month, SUM(s.amount) as total_sales, COUNT(*) as transaction_count, AVG(s.amount) as avg_sale_amount FROM sales s WHERE EXTRACT(YEAR FROM s.sale_date) = EXTRACT(YEAR FROM NOW()) GROUP BY DATE_TRUNC('month', s.sale_date) ORDER BY month"
    },
    {
      "key": "explanation",
      "value": "Aggregates sales data by month for the current year, showing total sales amount, transaction count, and average sale amount per month"
    },
    {
      "key": "confidence_score",
      "value": "0.93"
    },
    {
      "key": "execution_plan",
      "value": "Consider adding index on (sale_date) for optimal performance. Query will scan ~12 months of data."
    }
  ],
  "ai_info": {
    "processing_time_ms": 1456,
    "model_used": "codellama",
    "confidence_score": 0.93,
    "provider": "local"
  }
}
```

## Error Handling

### Error Response Format

```json
{
  "error": {
    "code": "AI_GENERATION_FAILED",
    "message": "Failed to generate SQL query from natural language input",
    "details": {
      "provider": "local",
      "model": "codellama",
      "confidence_threshold": 0.7,
      "actual_confidence": 0.4,
      "suggestion": "Try rephrasing your query or lowering the confidence threshold"
    }
  }
}
```

### Common Error Codes

| Code | Description | Resolution |
|------|-------------|------------|
| `AI_PROVIDER_UNAVAILABLE` | AI service is not accessible | Check provider configuration and connectivity |
| `AI_GENERATION_FAILED` | Failed to generate SQL | Rephrase query or adjust parameters |
| `CONFIDENCE_TOO_LOW` | Generated SQL below confidence threshold | Lower threshold or improve query description |
| `UNSUPPORTED_DATABASE` | Database type not supported | Use mysql, postgresql, or sqlite |
| `INVALID_CONFIGURATION` | Missing or invalid configuration | Check required properties |
| `RATE_LIMIT_EXCEEDED` | Too many requests | Wait before retrying |
| `MODEL_NOT_AVAILABLE` | Specified model not found | Check available models |
| `SQL_EXECUTION_DISABLED` | SQL execution not enabled | Enable in configuration |

### Error Handling Best Practices

1. **Check confidence scores** before executing generated SQL
2. **Validate generated SQL** in a safe environment first
3. **Implement retry logic** with exponential backoff for transient failures
4. **Monitor error rates** and adjust configuration accordingly
5. **Use fallback mechanisms** when AI generation fails

## Rate Limiting

The plugin implements rate limiting to prevent abuse and ensure fair usage:

### Default Limits

- **Per user**: 60 requests per minute
- **Global**: 600 requests per minute
- **Model-specific**: Varies by AI provider

### Rate Limit Headers

Response headers include rate limit information:

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1640995200
X-RateLimit-Retry-After: 60
```

### Rate Limit Exceeded Response

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Please wait before retrying.",
    "details": {
      "limit": 60,
      "remaining": 0,
      "reset_time": "2024-01-01T12:00:00Z",
      "retry_after": 60
    }
  }
}
```

## Authentication

### API Key Authentication

For cloud AI providers, API keys are configured server-side and not exposed to clients.

### Plugin Authentication

The plugin communicates with the main API Testing Tool via Unix socket, inheriting the main tool's authentication mechanism.

### Security Headers

All requests should include appropriate security headers:

```
Content-Type: application/json
User-Agent: atest-client/1.0
X-Request-ID: unique-request-identifier
```

## SDKs and Client Libraries

### Go Client Example

```go
package main

import (
    "context"
    "fmt"
    "google.golang.org/grpc"
    pb "github.com/linuxsuren/api-testing/pkg/testing/remote"
)

func main() {
    conn, err := grpc.Dial("unix:///tmp/atest-ext-ai.sock", grpc.WithInsecure())
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    client := pb.NewLoaderClient(conn)

    resp, err := client.Load(context.Background(), &pb.LoadRequest{
        Name: "ai",
        Query: "Find all active users created last month",
        Properties: []*pb.Pair{
            {Key: "ai_provider", Value: "local"},
            {Key: "model", Value: "codellama"},
            {Key: "database_type", Value: "mysql"},
        },
    })

    if err != nil {
        panic(err)
    }

    for _, data := range resp.Data {
        fmt.Printf("%s: %s\n", data.Key, data.Value)
    }
}
```

### Python Client Example

```python
import grpc
import json
from api_testing_pb2 import LoadRequest, Pair
from api_testing_pb2_grpc import LoaderStub

def generate_sql(query, database_type="mysql", provider="local", model="codellama"):
    with grpc.insecure_channel('unix:///tmp/atest-ext-ai.sock') as channel:
        client = LoaderStub(channel)

        request = LoadRequest(
            name="ai",
            query=query,
            properties=[
                Pair(key="ai_provider", value=provider),
                Pair(key="model", value=model),
                Pair(key="database_type", value=database_type),
            ]
        )

        response = client.Load(request)

        result = {}
        for data in response.data:
            result[data.key] = data.value

        result['ai_info'] = {
            'processing_time_ms': response.ai_info.processing_time_ms,
            'model_used': response.ai_info.model_used,
            'confidence_score': response.ai_info.confidence_score,
            'provider': response.ai_info.provider,
        }

        return result

# Usage
result = generate_sql("Find all products with price above 100")
print(json.dumps(result, indent=2))
```

### JavaScript/Node.js Client Example

```javascript
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');

const PROTO_PATH = 'path/to/loader.proto';
const packageDefinition = protoLoader.loadSync(PROTO_PATH);
const apiTesting = grpc.loadPackageDefinition(packageDefinition);

const client = new apiTesting.Loader(
    'unix:///tmp/atest-ext-ai.sock',
    grpc.credentials.createInsecure()
);

function generateSQL(query, options = {}) {
    return new Promise((resolve, reject) => {
        const request = {
            name: 'ai',
            query: query,
            properties: [
                { key: 'ai_provider', value: options.provider || 'local' },
                { key: 'model', value: options.model || 'codellama' },
                { key: 'database_type', value: options.databaseType || 'mysql' }
            ]
        };

        client.load(request, (error, response) => {
            if (error) {
                reject(error);
                return;
            }

            const result = {};
            response.data.forEach(item => {
                result[item.key] = item.value;
            });

            result.ai_info = {
                processing_time_ms: response.ai_info.processing_time_ms,
                model_used: response.ai_info.model_used,
                confidence_score: response.ai_info.confidence_score,
                provider: response.ai_info.provider
            };

            resolve(result);
        });
    });
}

// Usage
generateSQL('Find all active users created last month')
    .then(result => console.log(JSON.stringify(result, null, 2)))
    .catch(error => console.error('Error:', error));
```

## Monitoring and Metrics

### Prometheus Metrics

The plugin exposes metrics on port 9090:

- `atest_ai_requests_total{provider,model,status}` - Total AI requests
- `atest_ai_request_duration_seconds{provider,model}` - Request duration
- `atest_ai_confidence_score{provider,model}` - Confidence score distribution
- `atest_ai_active_connections` - Current active connections
- `atest_ai_provider_availability{provider}` - Provider availability status

### Health Checks

- **Liveness**: `/health/live` - Plugin is running
- **Readiness**: `/health/ready` - Plugin is ready to serve requests
- **Startup**: `/health/startup` - Plugin has started successfully

For more detailed monitoring setup, see the [Operations Guide](OPERATIONS.md).