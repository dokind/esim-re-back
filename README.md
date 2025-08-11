# eSIM Selling Platform

A comprehensive eSIM selling platform built with Go, PostgreSQL, and Docker. This platform integrates with RoamWiFi for eSIM provisioning and Mongolian QPay for payment processing.

## Features

- **eSIM Management**: Integration with RoamWiFi API for eSIM provisioning
- **Payment Processing**: Mongolian QPay integration for secure payments
- **Custom Pricing**: Set custom prices for eSIM packages
- **Order Management**: Complete order lifecycle management
- **User Management**: User registration, authentication, and profile management
- **Admin Panel**: Comprehensive admin interface for managing products, orders, and users
- **Analytics**: Sales and product analytics
- **Webhook Support**: Real-time payment notifications
- **Docker Support**: Complete containerization with docker-compose

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Go Backend    │    │   PostgreSQL    │
│   (Client)      │◄──►│   (Gin)         │◄──►│   Database      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Redis Cache   │    │   RoamWiFi API  │
                       └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   QPay API      │
                       │   (Mongolian)   │
                       └─────────────────┘
```

## Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)
- PostgreSQL 15+
- Redis 7+

## Quick Start

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd esim-re-back
   ```

2. **Set up environment variables**
   ```bash
   cp env.example .env
   # Edit .env with your configuration
   ```

3. **Start the services**
   ```bash
   docker-compose up -d
   ```

4. **Access the application**
   - API: http://localhost:8080
   - Health check: http://localhost:8080/health

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | 8080 |
| `DB_HOST` | PostgreSQL host | postgres |
| `DB_PORT` | PostgreSQL port | 5432 |
| `DB_USER` | Database user | esim_user |
| `DB_PASSWORD` | Database password | esim_password |
| `DB_NAME` | Database name | esim_db |
| `REDIS_HOST` | Redis host | redis |
| `REDIS_PORT` | Redis port | 6379 |
| `QPAY_MERCHANT_ID` | QPay merchant ID | - |
| `QPAY_MERCHANT_PASSWORD` | QPay merchant password | - |
| `QPAY_ENDPOINT` | QPay API endpoint | https://merchant.qpay.mn/v2 |
| `ROAMWIFI_API_KEY` | RoamWiFi API key | - |
| `ROAMWIFI_API_URL` | RoamWiFi API URL | - |
| `JWT_SECRET` | JWT signing secret | - |

### API Configuration

The platform provides the following API endpoints:

#### Authentication
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Token refresh

#### Products
- `GET /api/v1/products` - List all products
- `GET /api/v1/products/continents` - Products grouped by continent
- `GET /api/v1/products/:skuId/packages` - Packages for specific SKU
- `GET /api/v1/products/:id` - Get specific product

#### Orders
- `POST /api/v1/orders` - Create new order
- `GET /api/v1/orders/:orderNumber` - Get order details
- `POST /api/v1/orders/:orderNumber/pay` - Initiate payment

#### Webhooks
- `POST /api/v1/webhooks/qpay` - QPay payment webhook

#### Admin (Protected)
- `GET /api/v1/admin/products` - List all products (admin)
- `POST /api/v1/admin/products` - Create product (admin)
- `PUT /api/v1/admin/products/:id` - Update product (admin)
- `DELETE /api/v1/admin/products/:id` - Delete product (admin)
- `POST /api/v1/admin/products/sync` - Sync from RoamWiFi (admin)
- `GET /api/v1/admin/orders` - List all orders (admin)
- `GET /api/v1/admin/users` - List all users (admin)
- `GET /api/v1/admin/analytics/sales` - Sales analytics (admin)
- `GET /api/v1/admin/analytics/products` - Product analytics (admin)

## API Examples

### Create an Order

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "uuid-of-product",
    "customer_email": "customer@example.com",
    "customer_phone": "+97612345678",
    "custom_price": 15000
  }'
```

### Initiate Payment

```bash
curl -X POST http://localhost:8080/api/v1/orders/ESIM123456789/pay \
  -H "Content-Type: application/json"
```

### Get Products by Continent

```bash
curl -X GET http://localhost:8080/api/v1/products/continents
```

### Admin: Sync Products from RoamWiFi

```bash
curl -X POST http://localhost:8080/api/v1/admin/products/sync \
  -H "Authorization: Bearer your-admin-token"
```

## Database Schema

### Users
- `id` (UUID) - Primary key
- `email` (VARCHAR) - Unique email
- `password_hash` (VARCHAR) - Hashed password
- `first_name` (VARCHAR) - First name
- `last_name` (VARCHAR) - Last name
- `phone` (VARCHAR) - Phone number
- `is_admin` (BOOLEAN) - Admin flag
- `created_at` (TIMESTAMP) - Creation time
- `updated_at` (TIMESTAMP) - Update time

### Products
- `id` (UUID) - Primary key
- `sku_id` (VARCHAR) - RoamWiFi SKU ID
- `name` (VARCHAR) - Product name
- `description` (TEXT) - Product description
- `data_limit` (VARCHAR) - Data limit
- `validity_days` (INTEGER) - Validity in days
- `countries` (TEXT[]) - Array of country codes
- `continent` (VARCHAR) - Continent
- `base_price` (DECIMAL) - Base price
- `custom_price_usd` (DECIMAL) - Custom USD override price (optional; applied before FX conversion)
- `is_active` (BOOLEAN) - Active flag
- `created_at` (TIMESTAMP) - Creation time
- `updated_at` (TIMESTAMP) - Update time

### Orders
- `id` (UUID) - Primary key
- `user_id` (UUID) - User reference (optional)
- `product_id` (UUID) - Product reference
- `order_number` (VARCHAR) - Unique order number
- `qpay_invoice_id` (VARCHAR) - QPay invoice ID
- `status` (VARCHAR) - Order status
- `amount` (DECIMAL) - Order amount
- `currency` (VARCHAR) - Currency (default: MNT)
- `customer_email` (VARCHAR) - Customer email
- `customer_phone` (VARCHAR) - Customer phone
- `roamwifi_order_id` (VARCHAR) - RoamWiFi order ID
- `esim_data` (JSONB) - eSIM activation data
- `created_at` (TIMESTAMP) - Creation time
- `updated_at` (TIMESTAMP) - Update time

### Migration (2025-08-11)
Field rename: `custom_price` -> `custom_price_usd`.
If you had existing data run:
```sql
UPDATE products SET custom_price_usd = custom_price WHERE custom_price_usd IS NULL;
```
Then optionally drop old column if still present:
```sql
ALTER TABLE products DROP COLUMN IF EXISTS custom_price;
```

## Development

### Local Development Setup

1. **Install Go dependencies**
   ```bash
   go mod download
   ```

2. **Run database migrations**
   ```bash
   docker-compose up -d postgres redis
   # Wait for database to be ready
   go run cmd/server/main.go
   ```

3. **Run tests**
   ```bash
   go test ./...
   ```

### Docker Development

```bash
# Build and run all services
docker-compose up --build

# Run only specific services
docker-compose up -d postgres redis
docker-compose up app

# View logs
docker-compose logs -f app

# Stop all services
docker-compose down
```

## Deployment

### Production Deployment

1. **Set up SSL certificates**
   ```bash
   mkdir -p ssl
   # Add your SSL certificates to ssl/cert.pem and ssl/key.pem
   ```

2. **Configure environment variables**
   ```bash
   cp env.example .env
   # Edit .env with production values
   ```

3. **Deploy with Docker**
   ```bash
   docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
   ```

### Environment-Specific Configurations

- **Development**: Uses default docker-compose.yml
- **Production**: Use docker-compose.prod.yml for additional configurations
- **Staging**: Create docker-compose.staging.yml for staging environment

## Monitoring and Logging

### Health Checks
- Application: `GET /health`
- Database: Check PostgreSQL connection
- Redis: Check Redis connection

### Logging
- Application logs: `docker-compose logs app`
- Nginx logs: `docker-compose logs nginx`
- Database logs: `docker-compose logs postgres`

## Security Considerations

1. **Environment Variables**: Never commit sensitive data to version control
2. **SSL/TLS**: Always use HTTPS in production
3. **Rate Limiting**: Implemented at nginx level
4. **Input Validation**: All inputs are validated
5. **SQL Injection**: Using GORM with parameterized queries
6. **JWT Security**: Secure token generation and validation

## Troubleshooting

### Common Issues

1. **Database Connection Failed**
   - Check if PostgreSQL is running
   - Verify database credentials in .env
   - Check network connectivity

2. **QPay Integration Issues**
   - Verify QPay credentials
   - Check webhook URL configuration
   - Ensure proper SSL certificates

3. **RoamWiFi API Issues**
   - Verify API key and URL
   - Check API rate limits
   - Validate request format

### Debug Mode

Enable debug logging by setting:
```bash
LOG_LEVEL=debug
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Create an issue in the repository
- Contact the development team
- Check the documentation

## Roadmap

- [ ] Frontend web application
- [ ] Mobile app integration
- [ ] Advanced analytics dashboard
- [ ] Multi-language support
- [ ] Additional payment gateways
- [ ] Automated testing suite
- [ ] CI/CD pipeline
- [ ] Kubernetes deployment 