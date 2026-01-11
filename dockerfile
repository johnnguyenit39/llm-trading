# Stage 1: Build the Go application
FROM golang:1.23.4-alpine AS builder

# Set GOARCH to amd64 and GOOS to linux
ENV GOARCH=amd64
ENV GOOS=linux

WORKDIR /app

# Install build dependencies (e.g., git, if needed)
RUN apk add --no-cache git

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go application
RUN go build -o main .

# Stage 2: Create a lightweight production image
FROM alpine:latest

WORKDIR /app

# Install certificates for HTTPS communication if needed
RUN apk add --no-cache ca-certificates

# Expose port 80
EXPOSE 80

# Copy necessary files into the container
# Note: .env file is not copied as environment variables are set via build args/runtime env
COPY assets ./assets

# Copy the compiled binary from the builder stage
COPY --from=builder /app/main .

# Define the entry point
CMD ["./main"]
