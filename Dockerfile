# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install git, build tools and ca-certificates
RUN apk add --no-cache git ca-certificates build-base

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Install swag CLI for generating swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Generate Swagger docs (kept in repo path /docs)
RUN swag init -g cmd/server/main.go -o docs

# Build the application (after docs so swagger files are embedded via import)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Create logs directory
RUN mkdir -p /app/logs

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"] 