# Oil Price Scraper

```
   ____  _ _   ____       _           ____
  / __ \(_) | |  _ \ _ __(_) ___ ___ / ___|  ___ _ __ __ _ _ __   ___ _ __
 | |  | | | | | |_) | '__| |/ __/ _ \___ \ / __| '__/ _` | '_ \ / _ \ '__|
 | |__| | | | |  __/| |  | | (_|  __/___) | (__| | | (_| | |_) |  __/ |
  \____/|_|_| |_|   |_|  |_|\___\___|____/ \___|_|  \__,_| .__/ \___|_|
                                                         |_|
```

**Never miss a dip in heating oil prices again.**

A Go-based continuous scraper service that collects heating oil prices from multiple German APIs, stores them in PostgreSQL, and exposes Prometheus metrics for monitoring.

## Features

- **Multiple API Providers**: Supports HeizOel24 (nationwide average) and Hoyer (regional prices)
- **Daily Automated Scraping**: Built-in scheduler runs at a configurable hour each day
- **Historical Backfilling**: Import historical price data from supported APIs
- **Duplicate Prevention**: Automatically skips prices that already exist in the database
- **Prometheus Metrics**: Full observability with `/metrics` endpoint
- **Status Endpoint**: JSON status at `/status` for operational visibility
- **Structured Logging**: JSON or console output with zerolog
- **Docker Ready**: Multi-stage build with scratch runtime image
- **CI/CD Ready**: GitHub Actions for testing, linting, and Docker builds

## Quick Start

```bash
# Clone the repository
git clone https://github.com/andygrunwald/oil-price-scraper.git
cd oil-price-scraper

# Start with Docker Compose
docker-compose up -d

# Check the status
curl http://localhost:8080/status
```

## Installation

### Using Docker (Recommended)

```bash
docker pull ghcr.io/andygrunwald/oil-price-scraper:latest

docker run -d \
  -e POSTGRES_DSN="postgres://user:password@host:5432/oil?sslmode=disable" \
  -p 8080:8080 \
  ghcr.io/andygrunwald/oil-price-scraper:latest
```

### Building from Source

```bash
go install github.com/andygrunwald/oil-price-scraper/cmd/oilscraper@latest

# Or build manually
git clone https://github.com/andygrunwald/oil-price-scraper.git
cd oil-price-scraper
go build -o oilscraper ./cmd/oilscraper
```

## Usage

### Commands

```
oilscraper
  run       Start the continuous scraper service
  scrape    Run a one-time scrape
  backfill  Backfill historical data
  version   Print version information
```

### Run Command

Start the continuous scraper with daily scheduling:

```bash
oilscraper run \
  --postgres-dsn "postgres://user:password@localhost:5432/oil?sslmode=disable" \
  --zip-code "12345" \
  --scrape-hour 6 \
  --providers heizoel24,hoyer
```

### Scrape Command

Run a one-time scrape:

```bash
oilscraper scrape \
  --postgres-dsn "postgres://user:password@localhost:5432/oil?sslmode=disable" \
  --zip-code "12345" \
  --providers heizoel24,hoyer
```

### Backfill Command

Backfill historical data:

```bash
oilscraper backfill \
  --postgres-dsn "postgres://user:password@localhost:5432/oil?sslmode=disable" \
  --provider heizoel24 \
  --from 2024-01-01 \
  --to 2024-12-31
```

## Configuration

### Command-Line Flags

| Flag | Env Variable | Default | Description |
|------|--------------|---------|-------------|
| `--postgres-dsn` | `POSTGRES_DSN` | - | PostgreSQL connection string (required) |
| `--log-level` | `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `--log-format` | `LOG_FORMAT` | `json` | Log format (json, console) |
| `--store-raw-response` | `STORE_RAW_RESPONSE` | `true` | Store raw API responses |
| `--http-addr` | `HTTP_ADDR` | `:8080` | HTTP server address |
| `--zip-code` | `ZIP_CODE` | `47259` | Zip code for local price APIs |
| `--order-amount` | `ORDER_AMOUNT` | `3000` | Order amount in liters |

### Run Command Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--scrape-hour` | `6` | Hour of day (0-23) to scrape |
| `--providers` | `heizoel24,hoyer` | Comma-separated list of providers |

### Backfill Command Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--from` | - | Start date (YYYY-MM-DD, required) |
| `--to` | today | End date (YYYY-MM-DD) |
| `--provider` | `heizoel24` | Provider to backfill from |
| `--min-delay` | `1` | Minimum delay between requests (seconds) |
| `--max-delay` | `5` | Maximum delay between requests (seconds) |

## API Providers

### HeizOel24

- **Type**: Nationwide average price
- **Backfill Support**: Yes
- **API**: `https://www.heizoel24.de/api/chartapi/GetAveragePriceHistory`
- **Price Unit**: EUR per 100 liters (calculated for 3000L orders)

### Hoyer

- **Type**: Regional price (zip code specific)
- **Backfill Support**: No
- **API**: `https://api.hoyer.de/rest/heatingoil/{zipCode}/{amount}/{stations}`
- **Products**: Stores all available products (Bestpreis, Eco-Heizol, Express, etc.)
- **Note**: Requires browser-like User-Agent header

## HTTP Endpoints

### `/metrics` - Prometheus Metrics

Exposes Prometheus metrics including:

```
# API request metrics
oilscraper_api_requests_total{provider="heizoel24",status="success"}
oilscraper_api_request_duration_seconds{provider="heizoel24"}

# Price metrics
oilscraper_last_scrape_timestamp{provider="heizoel24"}
oilscraper_current_price_eur{provider="heizoel24",scope="national",product_type="standard"}

# Database metrics
oilscraper_db_operations_total{operation="insert",status="success"}
oilscraper_prices_stored_total{provider="heizoel24"}

# Standard Go runtime metrics
go_goroutines, go_memstats_*, etc.
```

### `/status` - Status Endpoint

Returns JSON with operational status:

```json
{
  "status": "healthy",
  "uptime_seconds": 86400,
  "next_scrape_at": "2026-01-13T06:00:00Z",
  "providers": {
    "heizoel24": {
      "enabled": true,
      "last_scrape_at": "2026-01-12T06:00:00Z",
      "last_scrape_success": true,
      "last_response_time_ms": 245,
      "last_price": 97.81,
      "total_requests": 365,
      "total_errors": 2
    }
  },
  "database": {
    "connected": true,
    "total_prices_stored": 1234
  }
}
```

### `/health` - Health Check

Returns `200 OK` if the service is running.

## Database Schema

```sql
CREATE TABLE oil_prices (
    id              BIGSERIAL PRIMARY KEY,
    provider        VARCHAR(50) NOT NULL,
    product_type    VARCHAR(50) NOT NULL DEFAULT 'standard',
    price_date      DATE NOT NULL,
    price_per_100l  DECIMAL(10, 4) NOT NULL,
    currency        VARCHAR(10) NOT NULL DEFAULT 'EUR',
    scope           VARCHAR(10) NOT NULL CHECK (scope IN ('local', 'national')),
    zip_code        VARCHAR(10) DEFAULT NULL,
    raw_response    JSONB DEFAULT NULL,
    fetched_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_provider_product_date UNIQUE NULLS NOT DISTINCT (provider, product_type, price_date, zip_code)
);

CREATE INDEX idx_price_date ON oil_prices (price_date);
CREATE INDEX idx_provider ON oil_prices (provider);
CREATE INDEX idx_product_type ON oil_prices (product_type);
```

## Development

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- PostgreSQL 18+ (or use Docker Compose)

### Local Development

```bash
# Start PostgreSQL
docker-compose up -d postgres

# Run the scraper locally
go run ./cmd/oilscraper run \
  --postgres-dsn "postgres://oilscraper:oilscraper@localhost:5432/oil?sslmode=disable" \
  --zip-code "12345" \
  --log-format console \
  --log-level debug

# Run tests
go test -v ./...

# Run linters
go vet ./...
golangci-lint run
staticcheck ./...

# Build
go build -o oilscraper ./cmd/oilscraper
```

### Docker Development

```bash
# Build and run everything
docker-compose up --build

# View logs
docker-compose logs -f oilscraper

# Access PostgreSQL
docker exec -it oilscraper-postgres psql -U oilscraper -d oil

# Query prices
docker exec -it oilscraper-postgres psql -U oilscraper -d oil \
  -c "SELECT * FROM oil_prices ORDER BY created_at DESC LIMIT 10;"
```

### Project Structure

```
oil-price-scraper/
├── cmd/oilscraper/          # CLI entry point
├── internal/
│   ├── api/                 # Provider interface
│   │   ├── heizoel24/       # HeizOel24 provider
│   │   └── hoyer/           # Hoyer provider
│   ├── config/              # Configuration
│   ├── database/            # PostgreSQL operations
│   ├── http/                # HTTP server & handlers
│   ├── models/              # Shared data types
│   ├── scheduler/           # Daily scheduler
│   ├── scraper/             # Scraping orchestration
│   └── useragent/           # User-Agent rotation
├── migrations/              # SQL schema
├── .github/workflows/       # CI/CD
├── Dockerfile
├── docker-compose.yml
└── README.md
```

### Releasing a New Version

1. Ensure all changes are committed and pushed to main
2. Create and push a version tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```
3. Goreleaser will automatically:
   - Build binaries for Linux and macOS (amd64/arm64)
   - Create a GitHub Release with changelog
   - Push Docker images to ghcr.io

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details.
