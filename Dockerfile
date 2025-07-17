# Build
FROM golang:alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum to download dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o /Stratum ./cmd/Stratum

# Run binary
FROM alpine:latest

WORKDIR /

# Copy the built binary from the builder stage
COPY --from=builder /Stratum /Stratum

# Expose the port the app runs on (default is 8080)
EXPOSE 8080

# Command to run the executable
CMD ["/Stratum"]
