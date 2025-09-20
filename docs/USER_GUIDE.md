# User Guide

This comprehensive guide covers advanced usage patterns, best practices, and real-world examples for the atest-ext-ai plugin.

## Table of Contents

- [Overview](#overview)
- [Advanced Query Patterns](#advanced-query-patterns)
- [Schema-Aware SQL Generation](#schema-aware-sql-generation)
- [Multi-Database Support](#multi-database-support)
- [AI Provider Comparison](#ai-provider-comparison)
- [Best Practices](#best-practices)
- [Real-World Examples](#real-world-examples)
- [Performance Optimization](#performance-optimization)
- [Integration Patterns](#integration-patterns)
- [Troubleshooting](#troubleshooting)

## Overview

The atest-ext-ai plugin transforms natural language descriptions into executable SQL queries, supporting multiple databases and AI providers. This guide helps you maximize its effectiveness in your testing workflows.

## Advanced Query Patterns

### Complex Aggregations

**Natural Language:** "Show quarterly revenue growth with percentage change and running totals"

**Generated SQL:**
```sql
WITH quarterly_sales AS (
    SELECT
        YEAR(order_date) as year,
        QUARTER(order_date) as quarter,
        SUM(total_amount) as revenue
    FROM orders
    GROUP BY YEAR(order_date), QUARTER(order_date)
),
revenue_with_growth AS (
    SELECT
        year,
        quarter,
        revenue,
        LAG(revenue) OVER (ORDER BY year, quarter) as prev_revenue,
        SUM(revenue) OVER (ORDER BY year, quarter ROWS UNBOUNDED PRECEDING) as running_total
    FROM quarterly_sales
)
SELECT
    year,
    quarter,
    revenue,
    CASE
        WHEN prev_revenue IS NOT NULL
        THEN ROUND(((revenue - prev_revenue) / prev_revenue) * 100, 2)
        ELSE NULL
    END as growth_percentage,
    running_total
FROM revenue_with_growth
ORDER BY year, quarter;
```

### Window Functions

**Natural Language:** "Rank products by sales within each category and show top 3"

**Generated SQL:**
```sql
WITH product_rankings AS (
    SELECT
        p.id,
        p.name,
        p.category,
        SUM(oi.quantity * oi.price) as total_sales,
        ROW_NUMBER() OVER (
            PARTITION BY p.category
            ORDER BY SUM(oi.quantity * oi.price) DESC
        ) as rank_in_category
    FROM products p
    JOIN order_items oi ON p.id = oi.product_id
    JOIN orders o ON oi.order_id = o.id
    WHERE o.status = 'completed'
    GROUP BY p.id, p.name, p.category
)
SELECT
    category,
    name,
    total_sales,
    rank_in_category
FROM product_rankings
WHERE rank_in_category <= 3
ORDER BY category, rank_in_category;
```

### Recursive Queries (PostgreSQL)

**Natural Language:** "Show employee hierarchy with all levels and path"

**Generated SQL:**
```sql
WITH RECURSIVE employee_hierarchy AS (
    -- Base case: top-level managers
    SELECT
        id,
        name,
        manager_id,
        1 as level,
        name::text as path
    FROM employees
    WHERE manager_id IS NULL

    UNION ALL

    -- Recursive case: employees with managers
    SELECT
        e.id,
        e.name,
        e.manager_id,
        eh.level + 1,
        eh.path || ' -> ' || e.name
    FROM employees e
    INNER JOIN employee_hierarchy eh ON e.manager_id = eh.id
)
SELECT
    id,
    name,
    level,
    path,
    REPEAT('  ', level - 1) || name as indented_name
FROM employee_hierarchy
ORDER BY path;
```

## Schema-Aware SQL Generation

### Providing Schema Context

To get better results, provide schema information in your requests:

```json
{
  "type": "ai",
  "natural_language": "Find customers who haven't placed orders in the last 6 months",
  "database_type": "mysql",
  "schema_context": {
    "tables": [
      {
        "name": "customers",
        "columns": [
          {"name": "id", "type": "INT", "primary_key": true},
          {"name": "email", "type": "VARCHAR(255)", "unique": true},
          {"name": "name", "type": "VARCHAR(255)"},
          {"name": "created_at", "type": "TIMESTAMP"}
        ]
      },
      {
        "name": "orders",
        "columns": [
          {"name": "id", "type": "INT", "primary_key": true},
          {"name": "customer_id", "type": "INT", "foreign_key": "customers.id"},
          {"name": "order_date", "type": "TIMESTAMP"},
          {"name": "status", "type": "ENUM('pending','completed','cancelled')"}
        ]
      }
    ],
    "relationships": [
      {
        "type": "one_to_many",
        "from": "customers.id",
        "to": "orders.customer_id"
      }
    ]
  }
}
```

**Generated Schema-Aware SQL:**
```sql
SELECT c.id, c.name, c.email, c.created_at
FROM customers c
LEFT JOIN orders o ON c.id = o.customer_id
    AND o.order_date >= DATE_SUB(NOW(), INTERVAL 6 MONTH)
WHERE o.id IS NULL
ORDER BY c.name;
```

### Schema Discovery

The plugin can work with various schema formats:

#### Database-Specific Information Schema

```json
{
  "schema_context": {
    "information_schema": {
      "database": "ecommerce",
      "tables": ["customers", "orders", "products", "order_items"]
    }
  }
}
```

#### Custom Schema Format

```json
{
  "schema_context": {
    "custom": {
      "customers": {
        "primary_key": "id",
        "columns": ["id", "email", "name", "created_at", "status"],
        "indexes": ["email", "status", "created_at"]
      },
      "orders": {
        "primary_key": "id",
        "foreign_keys": {
          "customer_id": "customers.id"
        },
        "columns": ["id", "customer_id", "order_date", "total_amount", "status"]
      }
    }
  }
}
```

## Multi-Database Support

### MySQL Specific Features

```json
{
  "type": "ai",
  "natural_language": "Show users with full-text search on bio containing 'developer'",
  "database_type": "mysql"
}
```

**Generated MySQL SQL:**
```sql
SELECT id, username, email, bio
FROM users
WHERE MATCH(bio) AGAINST('developer' IN BOOLEAN MODE)
ORDER BY MATCH(bio) AGAINST('developer' IN BOOLEAN MODE) DESC;
```

### PostgreSQL Specific Features

```json
{
  "type": "ai",
  "natural_language": "Find users with JSON data containing specific attributes",
  "database_type": "postgresql"
}
```

**Generated PostgreSQL SQL:**
```sql
SELECT id, username, email, metadata
FROM users
WHERE metadata->>'role' = 'admin'
   OR metadata->'preferences'->>'theme' = 'dark'
ORDER BY created_at DESC;
```

### SQLite Specific Features

```json
{
  "type": "ai",
  "natural_language": "Calculate running balance with date functions",
  "database_type": "sqlite"
}
```

**Generated SQLite SQL:**
```sql
SELECT
    id,
    transaction_date,
    amount,
    SUM(amount) OVER (
        ORDER BY datetime(transaction_date)
        ROWS UNBOUNDED PRECEDING
    ) as running_balance
FROM transactions
ORDER BY datetime(transaction_date);
```

## AI Provider Comparison

### Local Provider (Ollama)

**Best For:**
- Privacy-sensitive data
- Offline development
- Cost control
- Custom fine-tuned models

**Models:**
- `codellama`: Best for SQL generation
- `wizardcoder`: Complex query patterns
- `sqlcoder`: SQL-specific optimization

**Example Configuration:**
```yaml
ai:
  provider: local
  model: codellama
  local:
    ollama_endpoint: http://localhost:11434
    request_timeout: 120s
    max_retries: 3
```

**Pros:**
- No API costs
- Complete privacy
- Offline capability
- Customizable

**Cons:**
- Requires local resources
- Slower than cloud providers
- Model management needed

### OpenAI Provider

**Best For:**
- High accuracy requirements
- Complex query logic
- Fast response times
- Production environments

**Models:**
- `gpt-4`: Highest accuracy
- `gpt-3.5-turbo`: Fast and cost-effective

**Example Configuration:**
```yaml
ai:
  provider: openai
  model: gpt-4
  openai:
    api_key: ${OPENAI_API_KEY}
    max_tokens: 1000
    temperature: 0.2
```

**Pros:**
- High accuracy
- Fast responses
- No local setup
- Regular updates

**Cons:**
- API costs
- Internet required
- Data sent to OpenAI
- Rate limits

### Claude Provider (Anthropic)

**Best For:**
- Safety-critical applications
- Detailed explanations
- Code review integration
- Long context requirements

**Models:**
- `claude-3-opus`: Most capable
- `claude-3-sonnet`: Balanced
- `claude-3-haiku`: Fast

**Example Configuration:**
```yaml
ai:
  provider: claude
  model: claude-3-sonnet
  claude:
    api_key: ${CLAUDE_API_KEY}
    max_tokens: 1000
    temperature: 0.3
```

**Pros:**
- Safety-focused
- Detailed explanations
- Large context window
- Good reasoning

**Cons:**
- API costs
- Internet required
- Limited availability
- Rate limits

## Best Practices

### 1. Query Optimization Tips

#### Be Specific About Requirements
❌ **Vague:** "Get sales data"
✅ **Specific:** "Get monthly sales totals for the last 6 months with product categories"

#### Include Performance Hints
```json
{
  "natural_language": "Find top 10 customers by lifetime value with efficient indexing",
  "performance_hints": {
    "limit_results": true,
    "suggest_indexes": true,
    "avoid_full_scans": true
  }
}
```

#### Provide Sample Data Structure
```json
{
  "natural_language": "Calculate customer churn rate",
  "sample_data": {
    "customers": "id, email, subscription_start, subscription_end, status",
    "usage_logs": "customer_id, activity_date, action_type"
  }
}
```

### 2. Confidence Score Interpretation

```yaml
confidence_thresholds:
  0.9-1.0: "Excellent - Ready for production"
  0.8-0.89: "Good - Review before use"
  0.7-0.79: "Acceptable - Test thoroughly"
  0.6-0.69: "Poor - Needs refinement"
  0.0-0.59: "Unacceptable - Rephrase query"
```

### 3. Error Handling Strategies

```go
func handleAIResponse(response *pb.LoadResponse) error {
    confidenceScore := parseFloat(findValue(response.Data, "confidence_score"))

    if confidenceScore < 0.7 {
        return fmt.Errorf("confidence too low: %f", confidenceScore)
    }

    sql := findValue(response.Data, "generated_sql")
    if sql == "" {
        return fmt.Errorf("no SQL generated")
    }

    // Validate SQL syntax
    if err := validateSQL(sql); err != nil {
        return fmt.Errorf("invalid SQL: %w", err)
    }

    return nil
}
```

## Real-World Examples

### E-commerce Analytics

#### 1. Customer Segmentation
```json
{
  "natural_language": "Segment customers into high-value, medium-value, and low-value based on lifetime spend with percentile ranges"
}
```

**Generated SQL:**
```sql
WITH customer_stats AS (
    SELECT
        c.id,
        c.email,
        COALESCE(SUM(o.total_amount), 0) as lifetime_value,
        COUNT(o.id) as order_count
    FROM customers c
    LEFT JOIN orders o ON c.id = o.customer_id AND o.status = 'completed'
    GROUP BY c.id, c.email
),
percentiles AS (
    SELECT
        PERCENTILE_CONT(0.8) WITHIN GROUP (ORDER BY lifetime_value) as high_threshold,
        PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY lifetime_value) as medium_threshold
    FROM customer_stats
)
SELECT
    cs.*,
    CASE
        WHEN cs.lifetime_value >= p.high_threshold THEN 'High Value'
        WHEN cs.lifetime_value >= p.medium_threshold THEN 'Medium Value'
        ELSE 'Low Value'
    END as customer_segment
FROM customer_stats cs
CROSS JOIN percentiles p
ORDER BY cs.lifetime_value DESC;
```

#### 2. Inventory Optimization
```json
{
  "natural_language": "Find products with low stock that are selling well and need reordering"
}
```

**Generated SQL:**
```sql
SELECT
    p.id,
    p.name,
    p.current_stock,
    p.reorder_level,
    COALESCE(sales.units_sold_30days, 0) as recent_sales,
    COALESCE(sales.units_sold_30days, 0) / 30.0 as daily_sales_rate,
    CASE
        WHEN p.current_stock <= p.reorder_level
        AND COALESCE(sales.units_sold_30days, 0) > 0
        THEN 'URGENT_REORDER'
        WHEN p.current_stock <= p.reorder_level * 1.5
        AND COALESCE(sales.units_sold_30days, 0) > 10
        THEN 'NEEDS_REORDER'
        ELSE 'OK'
    END as reorder_status
FROM products p
LEFT JOIN (
    SELECT
        oi.product_id,
        SUM(oi.quantity) as units_sold_30days
    FROM order_items oi
    JOIN orders o ON oi.order_id = o.id
    WHERE o.status = 'completed'
      AND o.order_date >= DATE_SUB(NOW(), INTERVAL 30 DAY)
    GROUP BY oi.product_id
) sales ON p.id = sales.product_id
WHERE p.current_stock <= p.reorder_level * 2
ORDER BY
    CASE
        WHEN p.current_stock <= p.reorder_level
        AND COALESCE(sales.units_sold_30days, 0) > 0
        THEN 1
        ELSE 2
    END,
    sales.units_sold_30days DESC;
```

### Financial Reporting

#### 1. Monthly Recurring Revenue (MRR) Analysis
```json
{
  "natural_language": "Calculate MRR growth, churn, and net new MRR for SaaS business"
}
```

**Generated SQL:**
```sql
WITH monthly_subscriptions AS (
    SELECT
        DATE_FORMAT(billing_date, '%Y-%m') as month,
        customer_id,
        SUM(amount) as mrr
    FROM subscriptions
    WHERE status = 'active'
      AND billing_date >= DATE_SUB(CURDATE(), INTERVAL 12 MONTH)
    GROUP BY DATE_FORMAT(billing_date, '%Y-%m'), customer_id
),
mrr_by_month AS (
    SELECT
        month,
        SUM(mrr) as total_mrr,
        COUNT(DISTINCT customer_id) as active_customers
    FROM monthly_subscriptions
    GROUP BY month
),
mrr_changes AS (
    SELECT
        month,
        total_mrr,
        active_customers,
        LAG(total_mrr) OVER (ORDER BY month) as prev_mrr,
        total_mrr - LAG(total_mrr) OVER (ORDER BY month) as net_new_mrr
    FROM mrr_by_month
)
SELECT
    month,
    total_mrr,
    active_customers,
    prev_mrr,
    net_new_mrr,
    CASE
        WHEN prev_mrr > 0
        THEN ROUND((net_new_mrr / prev_mrr) * 100, 2)
        ELSE NULL
    END as growth_rate_percent
FROM mrr_changes
ORDER BY month;
```

### Operational Monitoring

#### 1. System Health Dashboard
```json
{
  "natural_language": "Show system health metrics with error rates and performance indicators"
}
```

**Generated SQL:**
```sql
SELECT
    DATE_FORMAT(timestamp, '%Y-%m-%d %H:00:00') as hour,
    service_name,
    COUNT(*) as total_requests,
    SUM(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 ELSE 0 END) as successful_requests,
    SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) as error_requests,
    ROUND(AVG(response_time_ms), 2) as avg_response_time,
    ROUND(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY response_time_ms), 2) as p95_response_time,
    ROUND(
        (SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) / COUNT(*)) * 100,
        2
    ) as error_rate_percent
FROM api_logs
WHERE timestamp >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
GROUP BY DATE_FORMAT(timestamp, '%Y-%m-%d %H:00:00'), service_name
HAVING COUNT(*) > 0
ORDER BY hour DESC, service_name;
```

## Performance Optimization

### 1. Caching Strategies

#### Enable Response Caching
```yaml
performance:
  cache:
    enabled: true
    ttl: 3600s  # Cache for 1 hour
    size: 100MB
```

#### Cache Key Considerations
- Include database type in cache key
- Consider schema version in cache key
- Use semantic hashing for similar queries

### 2. Request Batching

```go
// Batch multiple queries for efficiency
requests := []*pb.LoadRequest{
    {Query: "Get user count by status"},
    {Query: "Get average order value"},
    {Query: "Get top products by sales"},
}

responses := make(chan *pb.LoadResponse, len(requests))

for _, req := range requests {
    go func(r *pb.LoadRequest) {
        resp, err := client.Load(ctx, r)
        if err != nil {
            // Handle error
        }
        responses <- resp
    }(req)
}

// Collect results
for i := 0; i < len(requests); i++ {
    resp := <-responses
    // Process response
}
```

### 3. Connection Pooling

```yaml
performance:
  connection_pools:
    ai_providers:
      max_active_connections: 20
      max_idle_connections: 5
      connection_timeout: 30s
```

## Integration Patterns

### 1. Testing Pipeline Integration

```yaml
# .github/workflows/database-tests.yml
- name: Generate Test Queries
  run: |
    curl -X POST http://localhost:8080/api/v1/data/query \
      -H "Content-Type: application/json" \
      -d '{
        "type": "ai",
        "natural_language": "Create test data validation queries for user table",
        "database_type": "postgresql"
      }' > test_queries.json

- name: Execute Generated Tests
  run: |
    QUERIES=$(jq -r '.data[] | select(.key=="generated_sql") | .value' test_queries.json)
    psql $DATABASE_URL -c "$QUERIES"
```

### 2. API Documentation Generation

```python
def generate_api_examples():
    """Generate API documentation with real SQL examples"""

    example_queries = [
        "Get all active users",
        "Calculate monthly revenue",
        "Find top selling products"
    ]

    examples = {}

    for query in example_queries:
        response = ai_client.generate_sql(
            query=query,
            database_type="mysql"
        )

        examples[query] = {
            'sql': response['generated_sql'],
            'explanation': response['explanation'],
            'confidence': response['confidence_score']
        }

    return examples
```

### 3. Dynamic Query Building

```go
type QueryBuilder struct {
    client pb.LoaderClient
    config QueryConfig
}

func (qb *QueryBuilder) BuildQuery(intent string, context QueryContext) (*Query, error) {
    request := &pb.LoadRequest{
        Name: "ai",
        Query: intent,
        Properties: []*pb.Pair{
            {Key: "database_type", Value: context.DatabaseType},
            {Key: "confidence_threshold", Value: "0.8"},
        },
    }

    if context.Schema != nil {
        schemaJSON, _ := json.Marshal(context.Schema)
        request.Properties = append(request.Properties, &pb.Pair{
            Key:   "schema_context",
            Value: string(schemaJSON),
        })
    }

    resp, err := qb.client.Load(context.Background(), request)
    if err != nil {
        return nil, err
    }

    return parseQueryResponse(resp), nil
}
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Low Confidence Scores

**Problem:** Generated queries have confidence < 0.7

**Solutions:**
- Provide more specific natural language descriptions
- Include schema context
- Try different AI models
- Lower confidence threshold for testing

```yaml
ai:
  confidence_threshold: 0.6  # Lower threshold
  model: wizardcoder         # Try specialized model
```

#### 2. Incorrect SQL Syntax

**Problem:** Generated SQL doesn't match target database

**Solutions:**
- Always specify database type
- Provide database-specific examples
- Use schema context with column types

```json
{
  "database_type": "postgresql",
  "dialect_hints": {
    "use_limit": true,
    "date_functions": "postgresql",
    "string_functions": "postgresql"
  }
}
```

#### 3. Performance Issues

**Problem:** Slow AI response times

**Solutions:**
- Enable caching
- Use faster AI models
- Implement request batching
- Add connection pooling

```yaml
performance:
  cache:
    enabled: true
    ttl: 1800s

ai:
  model: gpt-3.5-turbo  # Faster than gpt-4
  request_timeout: 30s
```

#### 4. Schema Context Not Working

**Problem:** AI doesn't use provided schema information

**Solutions:**
- Verify schema format
- Include relationship information
- Use standard schema naming

```json
{
  "schema_context": {
    "version": "1.0",
    "tables": [
      {
        "name": "users",
        "columns": [
          {"name": "id", "type": "INTEGER", "constraints": ["PRIMARY_KEY"]},
          {"name": "email", "type": "VARCHAR(255)", "constraints": ["UNIQUE", "NOT_NULL"]}
        ]
      }
    ]
  }
}
```

### Debug Mode

Enable debug logging for troubleshooting:

```bash
export LOG_LEVEL=debug
atest-ext-ai --config config.yaml
```

Debug output includes:
- Raw AI provider requests/responses
- Schema parsing details
- Confidence calculation steps
- Performance metrics

### Getting Help

1. **Check logs** for detailed error messages
2. **Review configuration** for common mistakes
3. **Test with simple queries** first
4. **Verify AI provider connectivity**
5. **Check GitHub issues** for similar problems

For more troubleshooting information, see [TROUBLESHOOTING.md](TROUBLESHOOTING.md).