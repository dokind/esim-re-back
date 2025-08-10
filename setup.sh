#!/bin/bash

# =============================================================================
# eSIM Platform Setup Script
# =============================================================================

echo "🚀 Setting up eSIM Selling Platform..."

# Check if .env file exists
if [ -f ".env" ]; then
    echo "⚠️  .env file already exists. Backing up to .env.backup"
    cp .env .env.backup
fi

# Copy example.env to .env
echo "📋 Creating .env file from template..."
cp example.env .env

# Update QPay configuration with provided credentials
echo "🔧 Updating QPay configuration..."
sed -i '' 's/QPAY_USERNAME=your_qpay_username_here/QPAY_USERNAME=DOKIND_MN/g' .env
sed -i '' 's/QPAY_PASSWORD=your_qpay_password_here/QPAY_PASSWORD=xQF7fgDM/g' .env
sed -i '' 's/QPAY_INVOICE_CODE=your_invoice_code_here/QPAY_INVOICE_CODE=DOKIND_MN_INVOICE/g' .env
sed -i '' 's/QPAY_BASE_URL=your_base_url_here/QPAY_BASE_URL=https:\/\/merchant.qpay.mn/g' .env

echo "✅ QPay configuration updated with your credentials!"

# Install Go dependencies
echo "📦 Installing Go dependencies..."
go mod download

# Create necessary directories
echo "📁 Creating necessary directories..."
mkdir -p logs
mkdir -p ssl

echo "🔐 SSL certificates directory created. Add your certificates:"
echo "   - ssl/cert.pem (SSL certificate)"
echo "   - ssl/key.pem (SSL private key)"

echo ""
echo "🎯 Next steps:"
echo "1. Edit .env file with your actual credentials:"
echo "   - ROAMWIFI_API_KEY (get from RoamWiFi)"
echo "   - QPAY_MERCHANT_ID (get from QPay)"
echo "   - JWT_SECRET (generate a strong secret)"
echo "   - Update domain URLs"
echo ""
echo "2. Start the application:"
echo "   make docker-run"
echo ""
echo "3. Check the application:"
echo "   make health"
echo ""
echo "✅ Setup complete! 🎉" 