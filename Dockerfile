# Build stage
FROM golang:1.23-alpine AS builder

# Install git for go modules that need it
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with version information
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -o /oilscraper \
    ./cmd/oilscraper

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

# Copy the binary
COPY --from=builder /oilscraper /oilscraper

# Use non-root user for security
USER nonroot:nonroot

# Expose metrics/status port
EXPOSE 8080

ENTRYPOINT ["/oilscraper"]
CMD ["run"]
