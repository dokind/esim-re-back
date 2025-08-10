# eSIM Platform Admin Dashboard Specification

## Project Overview

Build a modern, responsive Next.js admin dashboard for managing an eSIM platform. This admin-only interface allows administrators to manage products, orders, users, pricing, and platform settings with full integration to the eSIM backend API.

## Tech Stack Requirements

- **Framework**: Next.js 14+ with App Router
- **Styling**: Tailwind CSS with shadcn/ui components
- **State Management**: Zustand or React Query (TanStack Query)
- **Authentication**: JWT tokens with refresh mechanism
- **API Integration**: Axios or Fetch API
- **Forms**: React Hook Form with Zod validation
- **Icons**: Lucide React or Heroicons
- **Deployment**: Vercel or similar

## Backend API Integration

**Base URL**: `http://localhost:8080/api/v1`

### Admin Authentication Endpoints

- `POST /auth/login` - Admin login (is_admin: true required)
- `POST /auth/refresh` - Token refresh
- `GET /user/profile` - Get admin profile

### Product Management Endpoints

- `GET /products/` - List all eSIM products with pagination
- `GET /products/continents` - Get products grouped by continent
- `GET /products/:id` - Get single product details
- `POST /admin/products/` - Create new product
- `PUT /admin/products/:id` - Update product
- `DELETE /admin/products/:id` - Delete product
- `POST /admin/products/sync` - Sync products from RoamWiFi API

### Order Management Endpoints

- `GET /admin/orders/` - Get all orders with filtering
- `GET /admin/orders/:id` - Get single order details
- `PUT /admin/orders/:id/status` - Update order status

### User Management Endpoints

- `GET /admin/users/` - Get all users
- `GET /admin/users/:id` - Get single user
- `PUT /admin/users/:id` - Update user details

### Settings & Analytics Endpoints

- `GET /admin/settings/` - Get admin settings
- `PUT /admin/settings/` - Update admin settings
- `GET /admin/pricing/info` - Get pricing information
- `PUT /admin/pricing/exchange-rate` - Update MNT exchange rate
- `POST /admin/pricing/update-all` - Update all product pricing
- `PUT /admin/products/:id/price` - Set custom product price
- `GET /admin/analytics/sales` - Get sales analytics
- `GET /admin/analytics/products` - Get product analytics

## Core Features

### 1. Authentication & Access Control

#### Admin Login Page (`/login`)

- **Single Purpose**: Admin-only login interface
- **Form Fields**: Email, Password
- **Validation**: Check for admin privileges (is_admin: true)
- **Security**: Rate limiting, secure token handling
- **Redirect**: Dashboard after successful login
- **Error Handling**: Clear error messages for failed login

### 2. Main Dashboard (`/dashboard`)

#### Overview Dashboard

- **Key Metrics Cards**:
  - Total Revenue (MNT) - today, this week, this month
  - Active Orders count
  - Total Customers count
  - Popular Destinations
- **Charts & Analytics**:
  - Revenue trends (line chart)
  - Sales by country/continent (bar chart)
  - Order status distribution (pie chart)
  - Monthly comparison metrics
- **Quick Actions**:
  - Sync Products from RoamWiFi
  - Create New Product
  - View Recent Orders
- **Recent Activity Feed**:
  - Latest orders
  - New customer registrations
  - System alerts and notifications

### 3. Product Management (`/products`)

#### Product List View

- **Data Table**: Sortable and filterable product list
  - SKU ID, Name, Country/Region
  - Data limit, Validity days
  - Base price, Custom price, MNT price
  - Status (Active/Inactive)
  - Last synced date
  - Actions (Edit, Delete, Toggle Status)
- **Bulk Actions**:
  - Enable/disable multiple products
  - Bulk price updates
  - Export product list
- **Search & Filters**:
  - Search by name or SKU
  - Filter by continent, status, price range
  - Sort by price, popularity, date added

#### Product Creation/Edit Modal

- **Basic Information**:
  - SKU ID, Product name, Description
  - Countries/regions covered
  - Continent selection
- **Technical Details**:
  - Data limit (1GB, 5GB, Unlimited, Custom)
  - Validity period (days)
  - Network compatibility
- **Pricing Management**:
  - Base price (USD)
  - Custom price override
  - Profit margin percentage
  - Calculated MNT price display
- **Status Control**: Active/inactive toggle

#### RoamWiFi Integration

- **Sync Products Button**: Large prominent button
- **Sync Status**: Show last sync time and status
- **Sync Progress**: Progress bar during sync operation
- **Sync History**: Log of previous sync operations
- **Manual Refresh**: Force refresh specific products

### 4. Order Management (`/orders`)

#### Order List View

- **Advanced Data Table**:
  - Order number, Customer info
  - Product details, Quantity
  - Amount (MNT), Payment status
  - Order status, Created date
  - Actions (View, Update Status, Contact Customer)
- **Status Management**:
  - Pending, Paid, Processing, Completed, Failed, Refunded
  - Bulk status updates
  - Status change history
- **Search & Filters**:
  - Date range picker
  - Status filter
  - Payment method filter
  - Customer search
  - Amount range filter

#### Order Detail View

- **Customer Information**:
  - Full customer details
  - Contact information
  - Order history
- **Order Details**:
  - Complete product information
  - Pricing breakdown
  - Payment information
  - eSIM delivery status
- **Order Actions**:
  - Update status
  - Send email to customer
  - Issue refund
  - Download eSIM details
  - Add internal notes

### 5. User Management (`/users`)

#### User List View

- **User Data Table**:
  - Name, Email, Phone
  - Registration date
  - Total orders, Total spent (MNT)
  - Account status (Active/Suspended)
  - Admin status
  - Actions (View, Edit, Suspend)
- **User Analytics**:
  - Customer lifetime value
  - Order frequency
  - Preferred destinations

#### User Detail View

- **Profile Information**: Personal details, contact info
- **Order History**: Complete order history with details
- **Account Actions**:
  - Suspend/activate account
  - Reset password
  - Send notifications
  - Upgrade to admin (if needed)

### 6. Settings & Configuration (`/settings`)

#### General Settings

- **Platform Configuration**:
  - Site name, description
  - Contact information
  - Support email/phone
  - Terms of service
- **Currency Settings**:
  - Default currency (MNT)
  - Exchange rate management
  - Rate update frequency

#### Payment Configuration

- **QPay Settings**:
  - Merchant ID configuration
  - API credentials management
  - Webhook endpoints
  - Test/Production mode toggle
- **Payment Options**:
  - Available payment methods
  - Transaction fees
  - Refund policies

#### RoamWiFi API Configuration

- **API Settings**:
  - API URL (testing vs production)
  - Phone number and password
  - API key management
  - Connection status
- **Sync Settings**:
  - Auto-sync frequency
  - Sync notifications
  - Error handling preferences

#### Pricing Management

- **Global Pricing Rules**:
  - Default profit margin percentage
  - Currency conversion rates
  - Pricing tiers by volume
- **Regional Pricing**:
  - Different margins by continent
  - Seasonal pricing adjustments
  - Promotional pricing rules

### 7. Analytics & Reports (`/analytics`)

#### Sales Analytics

- **Revenue Reports**:
  - Daily, weekly, monthly revenue
  - Revenue by product/country
  - Payment method breakdown
  - Refund and cancellation rates
- **Charts & Visualizations**:
  - Revenue trends over time
  - Top-selling products
  - Geographic sales distribution
  - Customer acquisition trends

#### Product Analytics

- **Product Performance**:
  - Best-selling products
  - Conversion rates by product
  - Price optimization insights
  - Inventory turnover
- **Customer Insights**:
  - Customer demographics
  - Purchase patterns
  - Repeat customer rate
  - Customer satisfaction metrics

#### Operational Reports

- **System Health**:
  - API response times
  - Error rates
  - Sync success rates
  - Payment gateway status
- **Export Options**:
  - CSV/Excel exports
  - Scheduled reports
  - Custom date ranges
  - Automated email reports

## UI/UX Requirements

### Design System

- **Colors**:
  - Primary: Professional blue (#1e40af)
  - Secondary: Slate gray (#64748b)
  - Success: Green (#22c55e)
  - Warning: Amber (#f59e0b)
  - Error: Red (#ef4444)
  - Background: Clean white/gray (#f8fafc)
- **Typography**: Inter or similar professional font
- **Spacing**: Consistent 8px grid system
- **Border Radius**: Subtle rounded corners (6px standard)
- **Shadows**: Subtle elevation for cards and modals

### Admin-Focused Design

- **Data Density**: High information density with good readability
- **Professional Layout**: Clean, business-focused design
- **Efficiency**: Quick access to common actions
- **Information Hierarchy**: Clear visual hierarchy for data
- **Consistency**: Consistent patterns across all pages

### Components to Build

- **Admin Header**: Logo, notifications, user menu, logout
- **Sidebar Navigation**: Collapsible menu with icons
- **Data Tables**: Sortable, filterable, paginated tables
- **Stats Cards**: KPI display cards with trend indicators
- **Charts**: Revenue, sales, and analytics charts
- **Modals**: For editing products, orders, settings
- **Form Components**: Professional form layouts
- **Loading States**: Skeleton loaders for data tables
- **Error States**: Professional error handling
- **Toast Notifications**: System feedback
- **Search & Filters**: Advanced filtering components
- **Bulk Actions**: Multi-select operations
- **Status Indicators**: Visual status displays

## Internationalization (Future)

- **Languages**: English (default), Mongolian
- **Currency**: MNT (Mongolian Tugrik) primary display
- **Date/Time**: Local formatting
- **RTL Support**: Consider for future expansion

## Security Requirements

- **Admin Authentication**: Strict admin-only access (is_admin: true)
- **JWT Handling**: Secure token storage and refresh
- **Role-Based Access**: Different admin permission levels
- **API Security**: Request/response validation
- **Input Sanitization**: Prevent XSS attacks
- **HTTPS**: Enforce secure connections
- **Session Management**: Automatic logout on inactivity
- **Audit Logging**: Track admin actions
- **Error Handling**: Don't expose sensitive information

## Performance Requirements

- **Page Load**: Under 3 seconds on 3G
- **SEO**: Server-side rendering for public pages
- **Images**: Optimized images with Next.js Image component
- **Caching**: Implement proper caching strategies
- **Analytics**: Google Analytics or similar tracking

## Testing Requirements

- **Unit Tests**: Jest and React Testing Library
- **Integration Tests**: API integration tests
- **E2E Tests**: Playwright or Cypress for critical flows
- **Accessibility**: WCAG 2.1 AA compliance

## Environment Configuration

```env
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_ADMIN_ONLY=true
```

## Key User Flows

### Admin Daily Operations Flow

1. Login to admin dashboard
2. Review daily metrics and alerts
3. Check new orders and update statuses
4. Monitor RoamWiFi sync status
5. Review customer issues and respond
6. Update pricing or product information
7. Generate reports for management

### Product Management Flow

1. Access product management section
2. Sync latest products from RoamWiFi API
3. Review new products and set pricing
4. Enable/disable products based on availability
5. Set custom pricing and profit margins
6. Monitor product performance metrics
7. Adjust pricing based on demand

### Order Processing Flow

1. Receive order notification
2. Verify payment status with QPay
3. Update order status to processing
4. Generate eSIM profile and QR code
5. Send delivery email to customer
6. Mark order as completed
7. Handle any customer support requests

### Settings Configuration Flow

1. Access admin settings
2. Update QPay payment configuration
3. Configure RoamWiFi API settings
4. Set exchange rates and pricing rules
5. Configure notification preferences
6. Save and test configuration changes
7. Monitor system health after changes

## Success Metrics

- **Admin Efficiency**: Time to complete common tasks
- **Data Accuracy**: Reduction in manual errors
- **System Uptime**: Platform availability and reliability
- **Order Processing**: Speed of order fulfillment
- **Revenue Management**: Pricing optimization effectiveness
- **User Satisfaction**: Admin user feedback scores
- **System Performance**: Dashboard load times and responsiveness

## Deliverables

1. **Complete Admin Dashboard**: All management features implemented
2. **Responsive Design**: Works on desktop, tablet, and mobile
3. **API Integration**: Full backend connectivity
4. **Data Visualization**: Charts and analytics dashboards
5. **Testing Suite**: Comprehensive test coverage
6. **Documentation**: Admin user guide and technical docs
7. **Performance Optimization**: Fast data loading and smooth UX

## Additional Considerations

- **Data Export**: Excel/CSV export functionality for reports
- **Bulk Operations**: Efficient bulk product and order management
- **Notification System**: Real-time alerts for important events
- **Backup & Recovery**: Data backup and system recovery procedures
- **Scalability**: Handle growing number of products and orders
- **Monitoring**: System health monitoring and alerting
- **Documentation**: Comprehensive admin user manual
- **Training**: Admin onboarding and training materials

This admin dashboard specification provides a complete foundation for building a professional eSIM platform management interface that enables efficient administration of the entire platform.
