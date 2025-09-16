-- Sample database initialization for atest-ext-ai plugin testing
-- This script creates sample tables and data for testing SQL generation

-- Drop existing tables if they exist
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS customers;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS suppliers;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS api_logs;

-- Users table for authentication/access testing
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    profile_data JSONB,
    preferences JSONB DEFAULT '{}'
);

-- Categories table
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    parent_id INTEGER REFERENCES categories(id),
    is_active BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Suppliers table
CREATE TABLE suppliers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    contact_email VARCHAR(255),
    contact_phone VARCHAR(50),
    address TEXT,
    country VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    rating DECIMAL(3,2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Products table
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    sku VARCHAR(100) UNIQUE NOT NULL,
    category_id INTEGER REFERENCES categories(id),
    supplier_id INTEGER REFERENCES suppliers(id),
    price DECIMAL(10,2) NOT NULL,
    cost DECIMAL(10,2),
    current_stock INTEGER DEFAULT 0,
    reorder_level INTEGER DEFAULT 10,
    weight DECIMAL(8,2),
    dimensions VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Customers table
CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(50),
    address TEXT,
    city VARCHAR(100),
    state VARCHAR(100),
    postal_code VARCHAR(20),
    country VARCHAR(100) DEFAULT 'USA',
    date_of_birth DATE,
    registration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'active',
    customer_type VARCHAR(20) DEFAULT 'regular',
    total_orders INTEGER DEFAULT 0,
    lifetime_value DECIMAL(12,2) DEFAULT 0.00,
    last_order_date TIMESTAMP,
    preferences JSONB DEFAULT '{}',
    marketing_consent BOOLEAN DEFAULT false
);

-- Orders table
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER REFERENCES customers(id),
    order_number VARCHAR(50) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    shipped_date TIMESTAMP,
    delivered_date TIMESTAMP,
    subtotal DECIMAL(10,2) NOT NULL,
    tax_amount DECIMAL(10,2) DEFAULT 0.00,
    shipping_amount DECIMAL(10,2) DEFAULT 0.00,
    discount_amount DECIMAL(10,2) DEFAULT 0.00,
    total_amount DECIMAL(10,2) NOT NULL,
    payment_method VARCHAR(50),
    payment_status VARCHAR(20) DEFAULT 'pending',
    shipping_address TEXT,
    billing_address TEXT,
    notes TEXT,
    tracking_number VARCHAR(100)
);

-- Order items table
CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10,2) NOT NULL,
    total_price DECIMAL(10,2) NOT NULL,
    discount_percent DECIMAL(5,2) DEFAULT 0.00
);

-- API logs table for monitoring and testing
CREATE TABLE api_logs (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    service_name VARCHAR(100),
    endpoint VARCHAR(255),
    method VARCHAR(10),
    status_code INTEGER,
    response_time_ms INTEGER,
    user_id INTEGER,
    ip_address INET,
    user_agent TEXT,
    request_body TEXT,
    response_body TEXT,
    error_message TEXT
);

-- Create indexes for better performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);

CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_supplier_id ON products(supplier_id);
CREATE INDEX idx_products_price ON products(price);
CREATE INDEX idx_products_stock ON products(current_stock);
CREATE INDEX idx_products_active ON products(is_active);

CREATE INDEX idx_customers_email ON customers(email);
CREATE INDEX idx_customers_status ON customers(status);
CREATE INDEX idx_customers_registration_date ON customers(registration_date);
CREATE INDEX idx_customers_lifetime_value ON customers(lifetime_value);

CREATE INDEX idx_orders_customer_id ON orders(customer_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_order_date ON orders(order_date);
CREATE INDEX idx_orders_total_amount ON orders(total_amount);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);

CREATE INDEX idx_api_logs_timestamp ON api_logs(timestamp);
CREATE INDEX idx_api_logs_service_status ON api_logs(service_name, status_code);

-- Insert sample data
INSERT INTO categories (name, description, sort_order) VALUES
('Electronics', 'Electronic devices and accessories', 1),
('Books', 'Books and educational materials', 2),
('Clothing', 'Apparel and accessories', 3),
('Home & Garden', 'Home improvement and garden supplies', 4),
('Sports', 'Sports equipment and accessories', 5);

INSERT INTO suppliers (name, contact_email, contact_phone, country, rating) VALUES
('TechSupplier Inc', 'orders@techsupplier.com', '+1-555-0101', 'USA', 4.5),
('BookDistributor LLC', 'sales@bookdist.com', '+1-555-0102', 'USA', 4.2),
('FashionWholesale Co', 'info@fashionwhole.com', '+1-555-0103', 'USA', 4.8),
('HomeGoods Supplier', 'contact@homegoods.com', '+1-555-0104', 'USA', 4.0),
('SportsPro Distributor', 'orders@sportspro.com', '+1-555-0105', 'USA', 4.7);

INSERT INTO products (name, description, sku, category_id, supplier_id, price, cost, current_stock, reorder_level) VALUES
('Laptop Computer', 'High-performance laptop for work and gaming', 'TECH-001', 1, 1, 1299.99, 999.99, 25, 10),
('Smartphone', 'Latest model smartphone with advanced features', 'TECH-002', 1, 1, 899.99, 699.99, 50, 15),
('Programming Book', 'Complete guide to modern software development', 'BOOK-001', 2, 2, 49.99, 29.99, 100, 20),
('Business Shirt', 'Professional dress shirt for office wear', 'CLOTH-001', 3, 3, 79.99, 39.99, 200, 50),
('Running Shoes', 'Comfortable shoes for jogging and running', 'SPORT-001', 5, 5, 149.99, 89.99, 75, 25),
('Garden Tools Set', 'Complete set of essential garden tools', 'HOME-001', 4, 4, 89.99, 49.99, 40, 15),
('Wireless Headphones', 'Noise-cancelling wireless headphones', 'TECH-003', 1, 1, 249.99, 149.99, 30, 10),
('Cookbook', 'Delicious recipes for home cooking', 'BOOK-002', 2, 2, 29.99, 19.99, 80, 20);

INSERT INTO customers (first_name, last_name, email, phone, city, state, country, customer_type, registration_date) VALUES
('John', 'Doe', 'john.doe@email.com', '+1-555-1001', 'New York', 'NY', 'USA', 'premium', CURRENT_TIMESTAMP - INTERVAL '6 months'),
('Jane', 'Smith', 'jane.smith@email.com', '+1-555-1002', 'Los Angeles', 'CA', 'USA', 'regular', CURRENT_TIMESTAMP - INTERVAL '4 months'),
('Bob', 'Johnson', 'bob.johnson@email.com', '+1-555-1003', 'Chicago', 'IL', 'USA', 'regular', CURRENT_TIMESTAMP - INTERVAL '3 months'),
('Alice', 'Williams', 'alice.williams@email.com', '+1-555-1004', 'Houston', 'TX', 'USA', 'premium', CURRENT_TIMESTAMP - INTERVAL '8 months'),
('Charlie', 'Brown', 'charlie.brown@email.com', '+1-555-1005', 'Phoenix', 'AZ', 'USA', 'regular', CURRENT_TIMESTAMP - INTERVAL '2 months'),
('Diana', 'Davis', 'diana.davis@email.com', '+1-555-1006', 'Philadelphia', 'PA', 'USA', 'premium', CURRENT_TIMESTAMP - INTERVAL '1 year'),
('Frank', 'Miller', 'frank.miller@email.com', '+1-555-1007', 'San Antonio', 'TX', 'USA', 'regular', CURRENT_TIMESTAMP - INTERVAL '5 months'),
('Grace', 'Wilson', 'grace.wilson@email.com', '+1-555-1008', 'San Diego', 'CA', 'USA', 'regular', CURRENT_TIMESTAMP - INTERVAL '7 months');

INSERT INTO users (username, email, password_hash, first_name, last_name, status) VALUES
('admin', 'admin@example.com', '$2a$10$hash1', 'Admin', 'User', 'active'),
('john_dev', 'john@dev.com', '$2a$10$hash2', 'John', 'Developer', 'active'),
('jane_tester', 'jane@test.com', '$2a$10$hash3', 'Jane', 'Tester', 'active'),
('bob_manager', 'bob@mgmt.com', '$2a$10$hash4', 'Bob', 'Manager', 'active'),
('inactive_user', 'inactive@example.com', '$2a$10$hash5', 'Inactive', 'User', 'inactive');

-- Insert sample orders
INSERT INTO orders (customer_id, order_number, status, order_date, subtotal, tax_amount, shipping_amount, total_amount, payment_method, payment_status) VALUES
(1, 'ORD-2024-001', 'completed', CURRENT_TIMESTAMP - INTERVAL '30 days', 1299.99, 104.00, 19.99, 1423.98, 'credit_card', 'completed'),
(2, 'ORD-2024-002', 'completed', CURRENT_TIMESTAMP - INTERVAL '25 days', 79.99, 6.40, 9.99, 96.38, 'paypal', 'completed'),
(3, 'ORD-2024-003', 'shipped', CURRENT_TIMESTAMP - INTERVAL '5 days', 249.99, 20.00, 14.99, 284.98, 'credit_card', 'completed'),
(1, 'ORD-2024-004', 'processing', CURRENT_TIMESTAMP - INTERVAL '2 days', 49.99, 4.00, 9.99, 63.98, 'credit_card', 'completed'),
(4, 'ORD-2024-005', 'completed', CURRENT_TIMESTAMP - INTERVAL '45 days', 899.99, 72.00, 19.99, 991.98, 'credit_card', 'completed');

-- Insert sample order items
INSERT INTO order_items (order_id, product_id, quantity, unit_price, total_price) VALUES
(1, 1, 1, 1299.99, 1299.99),  -- Laptop
(2, 4, 1, 79.99, 79.99),      -- Business Shirt
(3, 7, 1, 249.99, 249.99),    -- Wireless Headphones
(4, 3, 1, 49.99, 49.99),      -- Programming Book
(5, 2, 1, 899.99, 899.99);    -- Smartphone

-- Insert sample API logs for monitoring
INSERT INTO api_logs (service_name, endpoint, method, status_code, response_time_ms, user_id, ip_address) VALUES
('atest-ai-plugin', '/api/v1/data/query', 'POST', 200, 1250, 1, '192.168.1.100'),
('atest-ai-plugin', '/api/v1/data/query', 'POST', 200, 987, 2, '192.168.1.101'),
('atest-ai-plugin', '/metrics', 'GET', 200, 45, NULL, '192.168.1.102'),
('atest-ai-plugin', '/health', 'GET', 200, 12, NULL, '192.168.1.103'),
('atest-ai-plugin', '/api/v1/data/query', 'POST', 400, 123, 3, '192.168.1.104'),
('ollama', '/api/generate', 'POST', 200, 2340, NULL, '172.20.0.2'),
('ollama', '/api/tags', 'GET', 200, 67, NULL, '172.20.0.3');

-- Update customer statistics based on orders
UPDATE customers SET
    total_orders = (SELECT COUNT(*) FROM orders WHERE customer_id = customers.id),
    lifetime_value = (SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE customer_id = customers.id),
    last_order_date = (SELECT MAX(order_date) FROM orders WHERE customer_id = customers.id)
WHERE id IN (SELECT DISTINCT customer_id FROM orders);

-- Create a view for customer analytics
CREATE VIEW customer_analytics AS
SELECT
    c.id,
    c.first_name,
    c.last_name,
    c.email,
    c.customer_type,
    c.registration_date,
    c.total_orders,
    c.lifetime_value,
    c.last_order_date,
    CASE
        WHEN c.last_order_date >= CURRENT_TIMESTAMP - INTERVAL '30 days' THEN 'active'
        WHEN c.last_order_date >= CURRENT_TIMESTAMP - INTERVAL '90 days' THEN 'at_risk'
        ELSE 'churned'
    END as customer_segment,
    EXTRACT(DAYS FROM (CURRENT_TIMESTAMP - c.last_order_date))::INTEGER as days_since_last_order
FROM customers c;

-- Create a view for product performance
CREATE VIEW product_performance AS
SELECT
    p.id,
    p.name,
    p.sku,
    p.price,
    p.current_stock,
    c.name as category_name,
    s.name as supplier_name,
    COALESCE(sales.total_sold, 0) as total_sold,
    COALESCE(sales.total_revenue, 0) as total_revenue,
    COALESCE(sales.avg_order_quantity, 0) as avg_order_quantity
FROM products p
LEFT JOIN categories c ON p.category_id = c.id
LEFT JOIN suppliers s ON p.supplier_id = s.id
LEFT JOIN (
    SELECT
        oi.product_id,
        SUM(oi.quantity) as total_sold,
        SUM(oi.total_price) as total_revenue,
        AVG(oi.quantity) as avg_order_quantity
    FROM order_items oi
    JOIN orders o ON oi.order_id = o.id
    WHERE o.status = 'completed'
    GROUP BY oi.product_id
) sales ON p.id = sales.product_id;

-- Create a view for monthly sales summary
CREATE VIEW monthly_sales AS
SELECT
    DATE_TRUNC('month', order_date) as month,
    COUNT(*) as order_count,
    SUM(total_amount) as total_revenue,
    AVG(total_amount) as avg_order_value,
    COUNT(DISTINCT customer_id) as unique_customers
FROM orders
WHERE status = 'completed'
GROUP BY DATE_TRUNC('month', order_date)
ORDER BY month;

COMMIT;