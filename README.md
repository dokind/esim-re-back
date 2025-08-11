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
- `GET /api/v1/products` - List all products (basic info)
- `GET /api/v1/products/continents` - Products grouped by continent
- `GET /api/v1/products/skus` - List provider SKUs (public)
- `GET /api/v1/products/sku/:skuId` - Get a single SKU (public)
- `GET /api/v1/products/sku/:skuId/packages` - Packages for a specific SKU (optionally enriched)
- `GET /api/v1/products/:id` - Get specific product by internal UUID

#### Orders
- `POST /api/v1/orders` - Create new order (requires product_id + package_price_id or provider_price_id)
- `GET /api/v1/orders/:orderNumber` - Get order details
- `POST /api/v1/orders/:orderNumber/pay` - Initiate (or re-initiate) payment invoice

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

### Create an Order (User Purchase Flow)

```bash
curl -X POST http://localhost:8080/api/v1/orders \
   -H "Content-Type: application/json" \
   -d '{
      "product_id": "<product-uuid>",
      "package_price_id": "<package-price-uuid>",
      "customer_email": "customer@example.com",
      "customer_phone": "+97612345678",
      "custom_price_usd": 12.00
   }'
```

Notes:
- Provide either `package_price_id` (preferred) OR `provider_price_id` if you only have the upstream price id.
- `custom_price_usd` is optional; if omitted the system uses the package's effective USD price (base + markup or override).
- The server converts USD to MNT using the current stored exchange rate.

### Initiate Payment / Re-Issue Invoice

```bash
curl -X POST http://localhost:8080/api/v1/orders/ESIM123456789/pay \
   -H "Content-Type: application/json"
```

If the order already has a pending invoice the service may reuse or create a new one depending on state.

### Get Products by Continent

```bash
curl -X GET http://localhost:8080/api/v1/products/continents
```

### Admin: Sync Products from RoamWiFi

```bash
curl -X POST http://localhost:8080/api/v1/admin/products/sync \
  -H "Authorization: Bearer your-admin-token"
```

## User Purchase Flow (Detailed)

1. Discover packages:
   - `GET /api/v1/products/skus` or `GET /api/v1/products/sku/{skuId}/packages` (frontend obtains `package_price_id`, pricing metadata, validity, data limit, countries).
2. User selects a package; frontend stores `product_id` + `package_price_id`.
3. Create order (POST `/api/v1/orders`). Server:
   - Validates product & package alignment.
   - Determines USD price (override > markup > base provider price).
   - Converts to MNT.
   - Persists order (status=pending) + creates QPay invoice.
4. Response returns: order_number, amount (MNT), payment_url, qr_code.
5. User completes payment via QPay (browser / app).
6. QPay webhook (`POST /api/v1/webhooks/qpay`) marks order `paid` and triggers eSIM provisioning with RoamWiFi.
7. On success provisioning updates order to `completed` with eSIM activation data (QR / activation code).
8. Client polls `GET /api/v1/orders/{orderNumber}` (or future websocket) until status becomes `completed`.

Edge cases:
- Payment timeout → order stays `pending` or transitions to `expired` (future enhancement).
- Provider provisioning failure → order `failed`; manual intervention required.
- Re-initiate invoice if user lost it: POST `/api/v1/orders/{orderNumber}/pay`.

## Admin Pricing & Package Management Flow

1. Sync packages (manual or scheduled): `POST /api/v1/admin/products/sync` (or per-SKU sync endpoint) pulls provider SKUs & upserts `PackagePrice` records.
2. Adjust markup: `PUT /api/v1/admin/packages/{priceId}/markup { "percent": 15 }` recalculates effective USD + MNT.
3. Apply override: `PUT /api/v1/admin/packages/{priceId}/override { "price_usd": 9.99 }` (override supersedes markup).
4. Remove override: `PUT /api/v1/admin/packages/{priceId}/override` with null body or `DELETE` (if implemented) to revert to markup.
5. Update FX rate: `PUT /api/v1/admin/pricing/exchange-rate { "usd_to_mnt": 3450 }` then optionally `POST /api/v1/admin/pricing/update-all` to recompute MNT amounts.
6. Inspect pricing: (future) add an endpoint to list enriched package pricing for admin dashboards.

Operational notes:
- Orders always reference the `PackagePriceID` used at creation for historical price integrity.
- Changing markup/override after an order does not retroactively alter past orders.
- Use exchange rate refresh sparingly—bulk recompute ensures MNT consistency.

## Testing the Purchase Flow Locally

Minimal smoke test (requires valid env for QPay & RoamWiFi or stubs):
1. Start stack: `docker compose up -d --build`.
2. Sync products: `curl -X POST http://localhost:8080/api/v1/admin/products/sync -H "Authorization: Bearer <admin-token>"`.
3. Fetch packages for a SKU and pick one `package_price_id`.
4. Create order (as above) and open returned `payment_url`.
5. Simulate webhook (if no real QPay) by POSTing a crafted payload to `/api/v1/webhooks/qpay` matching the order_number; mark payment_status paid.
6. Get order and confirm status `completed` and presence of eSIM data fields.

If step 5 uses a manual webhook simulation ensure invoice IDs align or temporarily relax validation logic (development only).

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