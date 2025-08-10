-- Create database (PostgreSQL doesn't support IF NOT EXISTS for CREATE DATABASE)
-- The database is already created by the POSTGRES_DB environment variable
-- Connect to the database
\c esim_db;

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(20),
    is_admin BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create products table (eSIM packages)
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    data_limit VARCHAR(50),
    validity_days INTEGER,
    countries TEXT[], -- Array of country codes
    continent VARCHAR(50),
    base_price DECIMAL(10,2) NOT NULL,
    custom_price DECIMAL(10,2),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create orders table
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    product_id UUID REFERENCES products(id),
    order_number VARCHAR(100) UNIQUE NOT NULL,
    qpay_invoice_id VARCHAR(100),
    status VARCHAR(50) DEFAULT 'pending', -- pending, paid, processing, completed, failed
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'MNT',
    customer_email VARCHAR(255),
    customer_phone VARCHAR(20),
    roamwifi_order_id VARCHAR(100),
    esim_data JSONB, -- Store eSIM activation data
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create payment_transactions table
CREATE TABLE IF NOT EXISTS payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES orders(id),
    qpay_transaction_id VARCHAR(100),
    amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(50) NOT NULL,
    payment_method VARCHAR(50),
    transaction_data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create admin_settings table for custom pricing
CREATE TABLE IF NOT EXISTS admin_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    setting_key VARCHAR(100) UNIQUE NOT NULL,
    setting_value TEXT,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create audit_logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id UUID,
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
CREATE INDEX IF NOT EXISTS idx_products_sku_id ON products(sku_id);
CREATE INDEX IF NOT EXISTS idx_products_continent ON products(continent);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_order_id ON payment_transactions(order_id);

-- Insert default admin user (password: admin123)
INSERT INTO users (email, password_hash, first_name, last_name, is_admin) 
VALUES ('admin@esim.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'Admin', 'User', TRUE)
ON CONFLICT (email) DO NOTHING;

-- Insert default settings
INSERT INTO admin_settings (setting_key, setting_value, description) VALUES
('qpay_merchant_id', '', 'QPay Merchant ID'),
('qpay_merchant_password', '', 'QPay Merchant Password'),
('qpay_endpoint', '', 'QPay API Endpoint'),
('roamwifi_api_key', '', 'RoamWiFi API Key'),
('roamwifi_api_url', '', 'RoamWiFi API URL'),
('default_currency', 'MNT', 'Default currency for payments'),
('profit_margin_percentage', '10', 'Default profit margin percentage')
ON CONFLICT (setting_key) DO NOTHING; 