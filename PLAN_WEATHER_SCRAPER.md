# Plan: Add Weather Data Scraping Support

## Context

The oil-price-scraper application currently scrapes heating oil prices from two German providers (HeizOel24, Hoyer) using a clean provider-based architecture with Cobra CLI, PostgreSQL storage, Prometheus metrics, and daily scheduling.

We want to extend the application to also scrape **daily weather data** for a specific location (47259 Duisburg, ~51.4556°N 6.7623°E). The weather scraper should be a **separate binary** with the same command structure (`run`, `scrape`, `backfill`, `version`) and support **4 weather providers** initially: Open-Meteo, Bright Sky, Visual Crossing, and OpenWeather. DWD CDC-OpenData is deferred to a follow-up (Bright Sky already provides DWD data as JSON).

**Key decisions**:
- **Module name**: Keep `github.com/andygrunwald/oil-price-scraper` as-is (renaming is disruptive)
- **Architecture**: Option C — shared foundation, domain-specific layers. The scheduler gets an interface so both scrapers can use it. Database connection logic is shared. Everything domain-specific is new code in new packages. Existing oil code stays mostly untouched.
- **Rate limiting**: Simple delay between requests (--min-delay/--max-delay). User manages daily quotas. Log warning if backfill range exceeds ~900 days for rate-limited providers.
- **Coordinate precision**: Round lat/lon to 4 decimal places (~11m) before storage and uniqueness checks
- **DWD CDC-OpenData**: Deferred to follow-up — Bright Sky already wraps DWD data as JSON

---

## Milestone 1: Shared Infrastructure Refactoring

**Goal**: Make the scheduler reusable by both oil and weather scrapers.

### 1.1 Extract scheduler interface

**Create** `internal/scheduler/interface.go`:
```go
type ScraperInterface interface {
    ScrapeAll(ctx context.Context) error
    GetProviderNames() []string
    HasScrapedToday(ctx context.Context, providerName string) (bool, error)
    ScrapeProvider(ctx context.Context, providerName string) error
}
```

**Modify** `internal/scheduler/scheduler.go`:
- Change `scraper *scraper.Scraper` field to `scraper ScraperInterface`
- Change `New()` signature: `func New(s ScraperInterface, scrapeHour int, logger zerolog.Logger) *Scheduler`
- In `runIfNeeded()`: replace `s.scraper.GetProviders()` loop with `s.scraper.GetProviderNames()` loop and use `s.scraper.HasScrapedToday(ctx, name)` / `s.scraper.ScrapeProvider(ctx, name)`
- Remove import of `"github.com/andygrunwald/oil-price-scraper/internal/scraper"`

### 1.2 Add `GetProviderNames()` to existing oil scraper

**Modify** `internal/scraper/scraper.go`:
- Add method:
  ```go
  func (s *Scraper) GetProviderNames() []string {
      s.mu.RLock()
      defer s.mu.RUnlock()
      names := make([]string, 0, len(s.providers))
      for name := range s.providers {
          names = append(names, name)
      }
      return names
  }
  ```
- The existing `Scraper` already has `ScrapeAll`, `ScrapeProvider`, `HasScrapedToday` — it now satisfies `ScraperInterface` automatically.

### Verification
- Run `go build ./...` — both `cmd/oilscraper` and all tests must still compile
- Run `go test ./...` — all existing tests pass

---

## Milestone 2: Weather Data Model and Database Schema

**Goal**: Define weather data structures and database table.

### 2.1 Weather models

**Create** `internal/models/weather.go`:

```go
type WeatherResult struct {
    Date               time.Time
    Provider           string
    Latitude           float64
    Longitude          float64
    TemperatureMinC    *float64  // nullable — not all providers supply all fields
    TemperatureMaxC    *float64
    TemperatureMeanC   *float64
    PrecipitationMmSum *float64
    WindSpeedMaxKmh    *float64
    WindGustMaxKmh     *float64
    SunshineDurationS  *float64
    CloudCoverPercent  *float64
    HumidityPercent    *float64
    PressureHpa        *float64
    RawResponse        []byte
    FetchedAt          time.Time
}
```

Also add `WeatherProviderStatus` and `WeatherStatusResponse` types for the `/status` endpoint.

### 2.2 Database migration

**Create** `migrations/002_weather_schema.sql`:

```sql
CREATE TABLE IF NOT EXISTS weather_observations (
    id                    BIGSERIAL PRIMARY KEY,
    provider              VARCHAR(50) NOT NULL,
    observation_date      DATE NOT NULL,
    latitude              DECIMAL(9, 6) NOT NULL,
    longitude             DECIMAL(9, 6) NOT NULL,
    temperature_min_c     DECIMAL(6, 2) DEFAULT NULL,
    temperature_max_c     DECIMAL(6, 2) DEFAULT NULL,
    temperature_mean_c    DECIMAL(6, 2) DEFAULT NULL,
    precipitation_mm_sum  DECIMAL(8, 2) DEFAULT NULL,
    wind_speed_max_kmh    DECIMAL(6, 2) DEFAULT NULL,
    wind_gust_max_kmh     DECIMAL(6, 2) DEFAULT NULL,
    sunshine_duration_s   DECIMAL(10, 2) DEFAULT NULL,
    cloud_cover_percent   DECIMAL(5, 2) DEFAULT NULL,
    humidity_percent      DECIMAL(5, 2) DEFAULT NULL,
    pressure_hpa          DECIMAL(7, 2) DEFAULT NULL,
    raw_response          JSONB DEFAULT NULL,
    fetched_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_weather_provider_date_location
        UNIQUE (provider, observation_date, latitude, longitude)
);

CREATE INDEX IF NOT EXISTS idx_weather_observation_date ON weather_observations (observation_date);
CREATE INDEX IF NOT EXISTS idx_weather_provider ON weather_observations (provider);
CREATE INDEX IF NOT EXISTS idx_weather_location ON weather_observations (latitude, longitude);
```

### 2.3 Coordinate rounding helper

**Create** utility function (in `internal/models/weather.go` or a shared location):
```go
// RoundCoord rounds a coordinate to 4 decimal places (~11m precision).
func RoundCoord(v float64) float64 {
    return math.Round(v*10000) / 10000
}
```

All providers and the database layer round lat/lon through this function before storage and lookups.

### 2.4 Database operations

**Create** `internal/database/weather.go`:
- `InsertWeatherObservation(ctx, WeatherResult, storeRawResponse bool) error` — INSERT ON CONFLICT DO UPDATE (rounds lat/lon to 4 decimals before insert)
- `WeatherExistsForDate(ctx, provider string, date time.Time, lat, lon float64) (bool, error)` — rounds lat/lon before query
- `GetTotalWeatherCount(ctx) (int64, error)`

Same patterns as existing `InsertPrice` / `ExistsForDate` / `GetTotalPricesCount`.

### Verification
- Run migration against local PostgreSQL
- Run `go build ./...`

---

## Milestone 3: Weather Provider Interface and Scraper Orchestrator

**Goal**: Build the weather-domain equivalents of `internal/api/provider.go` and `internal/scraper/scraper.go`.

### 3.1 Weather provider interface

**Create** `internal/weatherapi/provider.go`:

```go
package weatherapi

type Provider interface {
    Name() string
    FetchCurrentWeather(ctx context.Context, lat, lon float64) ([]models.WeatherResult, error)
    FetchHistoricalWeather(ctx context.Context, lat, lon float64, from, to time.Time) ([]models.WeatherResult, error)
    SupportsBackfill() bool
    RequiresAPIKey() bool
}
```

### 3.2 Weather scraper orchestrator

**Create** `internal/weatherscraper/scraper.go`:

```go
type WeatherScraper struct {
    db               *database.DB
    providers        map[string]weatherapi.Provider
    providerMetrics  map[string]*Metrics
    promMetrics      PrometheusMetrics
    storeRawResponse bool
    latitude         float64
    longitude        float64
    logger           zerolog.Logger
    mu               sync.RWMutex
}
```

Methods (mirror the oil scraper pattern):
- `New(db, storeRaw, lat, lon, logger) *WeatherScraper`
- `RegisterProvider(provider weatherapi.Provider)`
- `ScrapeAll(ctx) error`
- `ScrapeProvider(ctx, name) error`
- `Backfill(ctx, name, from, to, minDelay, maxDelay) error`
- `HasScrapedToday(ctx, name) (bool, error)`
- `GetProviderNames() []string` — satisfies `scheduler.ScraperInterface`
- `GetProviders() []weatherapi.Provider`
- `GetMetrics(name) *Metrics`
- `SetPrometheusMetrics(m PrometheusMetrics)`

The `Metrics` / `MetricsSnapshot` types can be reused from the oil scraper or duplicated (they're small). I recommend duplicating to avoid coupling: `internal/weatherscraper/metrics.go`.

`PrometheusMetrics` interface for weather:
```go
type PrometheusMetrics interface {
    RecordAPIRequest(provider, status string, duration float64)
    RecordLastScrape(provider string, timestamp float64)
    RecordCurrentTemperature(provider string, temp float64)
    RecordDBOperation(operation, status string)
    RecordObservationsStored(provider string, count float64)
}
```

### 3.3 Weather config

**Create** `internal/weatherconfig/config.go`:

```go
type Config struct {
    PostgresDSN          string
    LogLevel             string
    LogFormat            string
    StoreRawResponse     bool
    HTTPAddr             string
    Latitude             float64
    Longitude            float64
    ScrapeHour           int
    Providers            []string
    VisualCrossingAPIKey string
    OpenWeatherAPIKey    string
    Backfill             BackfillConfig
}
```

Environment variables: `POSTGRES_DSN`, `LOG_LEVEL`, `LOG_FORMAT`, `STORE_RAW_RESPONSE`, `HTTP_ADDR`, `LATITUDE`, `LONGITUDE`, `SCRAPE_HOUR`, `PROVIDERS`, `VISUAL_CROSSING_API_KEY`, `OPENWEATHER_API_KEY`.

### Verification
- `go build ./...` compiles
- Unit test the WeatherScraper with a mock provider

---

## Milestone 4: Implement Weather Providers

Implement in order of increasing complexity. Each provider gets its own sub-package under `internal/weatherapi/`.

### 4a. Open-Meteo (simplest — validate end-to-end first)

**Create** `internal/weatherapi/openmeteo/openmeteo.go`

| Aspect | Detail |
|--------|--------|
| Auth | None (free tier, no API key) |
| Backfill | Yes — historical back to 1940 (ERA5 reanalysis) |
| Daily aggregates | Provided directly via `daily=` parameter |
| Forecast endpoint | `GET https://api.open-meteo.com/v1/forecast?latitude=X&longitude=Y&daily=temperature_2m_max,temperature_2m_min,temperature_2m_mean,precipitation_sum,wind_speed_10m_max,wind_gusts_10m_max,sunshine_duration,cloud_cover_mean,relative_humidity_2m_mean,surface_pressure_mean&timezone=Europe/Berlin` |
| Archive endpoint | `GET https://archive-api.open-meteo.com/v1/archive?latitude=X&longitude=Y&start_date=YYYY-MM-DD&end_date=YYYY-MM-DD&daily=...&timezone=Europe/Berlin` |
| Response | JSON with `daily.time[]` (dates) and parallel arrays for each variable |
| Rate limits | 10,000 calls/day, 5,000/hour, 600/minute (free tier) |
| Notes | `timezone` parameter is **required** when using `daily`. Use `Europe/Berlin` for correct day boundaries. |

Implementation:
- `FetchCurrentWeather`: Use forecast endpoint with `forecast_days=1` or `past_days=1&forecast_days=1`
- `FetchHistoricalWeather`: Use archive endpoint with date range (can handle large ranges in one call)
- Parse parallel arrays: `daily.time[i]` corresponds to `daily.temperature_2m_max[i]`, etc.

### 4b. Bright Sky (DWD data as JSON)

**Create** `internal/weatherapi/brightsky/brightsky.go`

| Aspect | Detail |
|--------|--------|
| Auth | None |
| Backfill | Yes — from 2010+ |
| Daily aggregates | **No** — returns hourly data, must aggregate client-side |
| Endpoint | `GET https://api.brightsky.dev/weather?lat=X&lon=Y&date=YYYY-MM-DD&last_date=YYYY-MM-DD` |
| Response | JSON with `weather[]` array of hourly records (timestamp, temperature, precipitation, wind_speed, sunshine, cloud_cover, etc.) |
| Rate limits | No explicit per-user limits |

Implementation:
- `FetchCurrentWeather`: Fetch today's hourly data, aggregate to one daily record
- `FetchHistoricalWeather`: Fetch date range (API supports multi-day), aggregate per day
- **Aggregation logic** (key part): Group hourly records by date, then:
  - `temperature_min_c` = min of hourly `temperature`
  - `temperature_max_c` = max of hourly `temperature`
  - `temperature_mean_c` = mean of hourly `temperature`
  - `precipitation_mm_sum` = sum of hourly `precipitation`
  - `wind_speed_max_kmh` = max of hourly `wind_speed`
  - `wind_gust_max_kmh` = max of hourly `wind_gust_speed`
  - `sunshine_duration_s` = sum of hourly `sunshine` (minutes → seconds)
  - `cloud_cover_percent` = mean of hourly `cloud_cover`
  - `humidity_percent` = mean of hourly `relative_humidity`
  - `pressure_hpa` = mean of hourly `pressure_msl`

Consider creating a helper: `aggregateHourlyToDaily(records []hourlyRecord) []models.WeatherResult`

### 4c. Visual Crossing

**Create** `internal/weatherapi/visualcrossing/visualcrossing.go`

| Aspect | Detail |
|--------|--------|
| Auth | API key required (free signup, 1000 records/day) |
| Backfill | Yes |
| Daily aggregates | Provided directly |
| Endpoint | `GET https://weather.visualcrossing.com/VisualCrossingWebServices/rest/services/timeline/{lat},{lon}/{startDate}/{endDate}?key={KEY}&include=days&unitGroup=metric&contentType=json` |
| Response | JSON with `days[]` array, each with `datetime`, `tempmax`, `tempmin`, `temp`, `precip`, `windspeed`, `windgust`, `cloudcover`, `humidity`, `pressure`, `uvindex`, etc. |
| Rate limits | 1000 records/day free tier |

Implementation:
- `FetchCurrentWeather`: Timeline API with today's date
- `FetchHistoricalWeather`: Timeline API with date range
- Map response fields: `tempmax` → `TemperatureMaxC`, `tempmin` → `TemperatureMinC`, `temp` → `TemperatureMeanC`, `precip` → `PrecipitationMmSum`, `windspeed` → `WindSpeedMaxKmh`, `windgust` → `WindGustMaxKmh`, etc.
- Handle `queryCost` field for rate limit awareness

### 4d. OpenWeather One Call 3.0

**Create** `internal/weatherapi/openweather/openweather.go`

| Aspect | Detail |
|--------|--------|
| Auth | API key required (`appid` parameter) |
| Backfill | Yes — from 1979+ |
| Daily aggregates | Provided directly via `day_summary` endpoint |
| Endpoint | `GET https://api.openweathermap.org/data/3.0/onecall/day_summary?lat=X&lon=Y&date=YYYY-MM-DD&appid={KEY}&units=metric` |
| Response | JSON with `temperature.min/max/morning/afternoon/evening/night`, `precipitation.total`, `wind.max.speed/direction`, `humidity.afternoon`, `pressure.afternoon`, `cloud_cover.afternoon` |
| Rate limits | 1000 calls/day free |
| **Critical**: ONE day per request | Backfill of N days = N API calls |

Implementation:
- `FetchCurrentWeather`: Single call for today
- `FetchHistoricalWeather`: **Loop day-by-day** with sleep between requests
  - Accept delay config via constructor: `New(logger, apiKey, minDelay, maxDelay)`
  - For each day in range: fetch, parse, append result, sleep
- Map response: `temperature.min` → `TemperatureMinC`, `temperature.max` → `TemperatureMaxC`, calculate mean from `(min+max)/2` or `(morning+afternoon+evening+night)/4`, `precipitation.total` → `PrecipitationMmSum`, `wind.max.speed` → `WindSpeedMaxKmh` (convert m/s to km/h: ×3.6)
- **Unit conversion**: OpenWeather `wind.max.speed` is in m/s with `units=metric`; convert to km/h for consistency

### 4e. DWD CDC-OpenData — DEFERRED

Deferred to a follow-up. Bright Sky already provides DWD station data as a clean JSON REST API. The DWD CDC direct integration (ZIP downloads, fixed-width parsing, station discovery) adds significant complexity for marginal benefit. The provider interface is designed so DWD CDC can be added later without changes to the rest of the system.

### Provider implementation order
1. Open-Meteo → validate full pipeline end-to-end
2. Bright Sky → test hourly aggregation
3. Visual Crossing → test API key handling
4. OpenWeather → test day-by-day backfill with rate limiting

### Verification per provider
- Unit tests with fixture response data (save example JSON/text in `testdata/` dirs)
- Integration smoke test: `weatherscraper scrape --providers=openmeteo`

---

## Milestone 5: Weather Scraper Binary

**Goal**: Build `cmd/weatherscraper/` with all 4 commands.

### 5.1 Weather HTTP server and metrics

**Create** `internal/weatherhttp/server.go`:
- Same pattern as `internal/http/server.go`
- Endpoints: `/metrics`, `/status`, `/health`
- Wires `WeatherScraper`, `Scheduler`, `DB`

**Create** `internal/weatherhttp/metrics.go`:
- Prometheus metrics with `weatherscraper_` prefix:
  - `weatherscraper_api_requests_total{provider, status}`
  - `weatherscraper_api_request_duration_seconds{provider}`
  - `weatherscraper_last_scrape_timestamp{provider}`
  - `weatherscraper_current_temperature_celsius{provider}`
  - `weatherscraper_db_operations_total{operation, status}`
  - `weatherscraper_observations_stored_total{provider}`

**Create** `internal/weatherhttp/status.go`:
- Weather-specific status response

### 5.2 CLI commands

**Create** `cmd/weatherscraper/main.go`:
- Root command: `weatherscraper`
- Global flags:
  - `--postgres-dsn` (shared)
  - `--log-level` (shared)
  - `--log-format` (shared)
  - `--store-raw-response` (shared)
  - `--http-addr` (shared, default `:8081` to avoid conflict with oil scraper)
  - `--latitude` (weather-specific, required)
  - `--longitude` (weather-specific, required)

**Create** `cmd/weatherscraper/cmd_run.go`:
- Flags: `--scrape-hour` (default 7), `--providers` (default "openmeteo"), `--visual-crossing-api-key`, `--openweather-api-key`
- Provider registration:
  ```go
  switch p {
  case "openmeteo":
      s.RegisterProvider(openmeteo.New(logger))
  case "brightsky":
      s.RegisterProvider(brightsky.New(logger))
  case "visualcrossing":
      s.RegisterProvider(visualcrossing.New(logger, cfg.VisualCrossingAPIKey))
  case "openweather":
      s.RegisterProvider(openweather.New(logger, cfg.OpenWeatherAPIKey))
  }
  ```
- Same signal handling, graceful shutdown pattern as oil scraper

**Create** `cmd/weatherscraper/cmd_scrape.go`:
- One-time scrape, same pattern

**Create** `cmd/weatherscraper/cmd_backfill.go`:
- Flags: `--from`, `--to`, `--provider`, `--min-delay`, `--max-delay`
- Same pattern as oil backfill

**Create** `cmd/weatherscraper/cmd_version.go`:
- Same pattern as oil version

### Verification
- `go build ./cmd/weatherscraper` produces `weatherscraper` binary
- `weatherscraper scrape --postgres-dsn=... --latitude=51.4556 --longitude=6.7623 --providers=openmeteo` works end-to-end
- `weatherscraper run` starts scheduler and HTTP server
- `weatherscraper backfill --from=2024-01-01 --to=2024-01-31 --provider=openmeteo` works

---

## Milestone 6: Docker and Deployment Updates

### 6.1 Makefile updates

**Modify** `Makefile`:
- Add `build-weather` target
- Add `build-all` target (builds both binaries)
- Keep existing `build` target for oil (backwards compat)

### 6.2 GoReleaser updates

**Modify** `.goreleaser.yaml`:
- Add second build entry for `weatherscraper` binary (same platforms/flags)
- Add second Docker image: `ghcr.io/andygrunwald/weather-scraper`

### 6.3 Docker updates

**Create** `Dockerfile.weather`:
- Same pattern as existing Dockerfile but copies `weatherscraper` binary
- Default port: 8081

**Modify** `docker-compose.yml`:
- Add `weatherscraper` service:
  - Depends on PostgreSQL
  - Environment: `POSTGRES_DSN`, `LATITUDE=51.4556`, `LONGITUDE=6.7623`, `PROVIDERS=openmeteo`, etc.
  - Port: 8081
- Migration 002 is auto-applied (already mounted from `./migrations/`)

### Verification
- `docker compose up` starts PostgreSQL, oilscraper, and weatherscraper
- Both `/status` endpoints respond (port 8080 for oil, 8081 for weather)

---

## Milestone 7: Testing

### Unit tests per provider
- `internal/weatherapi/openmeteo/openmeteo_test.go` — response parsing
- `internal/weatherapi/brightsky/brightsky_test.go` — response parsing + hourly→daily aggregation
- `internal/weatherapi/visualcrossing/visualcrossing_test.go` — response parsing
- `internal/weatherapi/openweather/openweather_test.go` — response parsing, unit conversion

### Orchestrator tests
- `internal/weatherscraper/scraper_test.go` — mock provider, verify flow

### Database tests
- `internal/database/weather_test.go` — insert, exists, count operations

### Test fixture data
- `internal/weatherapi/openmeteo/testdata/response.json`
- `internal/weatherapi/brightsky/testdata/response.json`
- `internal/weatherapi/visualcrossing/testdata/response.json`
- `internal/weatherapi/openweather/testdata/response.json`

---

## Files Summary

### New files (~25 files)
| File | Purpose |
|------|---------|
| `internal/scheduler/interface.go` | ScraperInterface for scheduler reuse |
| `internal/models/weather.go` | WeatherResult, WeatherObservation, status types |
| `migrations/002_weather_schema.sql` | weather_observations table |
| `internal/database/weather.go` | Weather DB operations |
| `internal/weatherapi/provider.go` | Weather Provider interface |
| `internal/weatherapi/openmeteo/openmeteo.go` | Open-Meteo provider |
| `internal/weatherapi/brightsky/brightsky.go` | Bright Sky provider |
| `internal/weatherapi/visualcrossing/visualcrossing.go` | Visual Crossing provider |
| `internal/weatherapi/openweather/openweather.go` | OpenWeather provider |
| `internal/weatherscraper/scraper.go` | Weather scraper orchestrator |
| `internal/weatherscraper/metrics.go` | Weather scraper metrics types |
| `internal/weatherconfig/config.go` | Weather-specific config |
| `internal/weatherhttp/server.go` | Weather HTTP server |
| `internal/weatherhttp/metrics.go` | Weather Prometheus metrics |
| `internal/weatherhttp/status.go` | Weather status handler |
| `cmd/weatherscraper/main.go` | Weather CLI entry point |
| `cmd/weatherscraper/cmd_run.go` | Weather run command |
| `cmd/weatherscraper/cmd_scrape.go` | Weather scrape command |
| `cmd/weatherscraper/cmd_backfill.go` | Weather backfill command |
| `cmd/weatherscraper/cmd_version.go` | Weather version command |
| `Dockerfile.weather` | Weather Docker image |
| + test files and testdata | See Milestone 7 |

### Modified files (~5 files)
| File | Change |
|------|--------|
| `internal/scheduler/scheduler.go` | Use `ScraperInterface` instead of `*scraper.Scraper` |
| `internal/scraper/scraper.go` | Add `GetProviderNames()` method |
| `Makefile` | Add weather build targets |
| `.goreleaser.yaml` | Add weatherscraper build + Docker |
| `docker-compose.yml` | Add weatherscraper service |

---

## Implementation Sequence

```
M1 (scheduler refactoring) ──┐
                              ├──→ M3 (interface + orchestrator) ──→ M4a (Open-Meteo) ──→ M5 (binary) ──┐
M2 (models + migration) ─────┘                                                                          │
                                                                                                         ├──→ M6 (Docker) ──→ M7 (tests)
                                                              M4b (Bright Sky) ──────────────────────────┤
                                                              M4c (Visual Crossing) ────────────────────┤
                                                              M4d (OpenWeather) ────────────────────────┘
```

**Critical path**: M1+M2 → M3 → M4a → M5 (validates full pipeline). Then remaining providers (M4b-4d) can be implemented incrementally.

**Follow-up** (not in this plan): DWD CDC-OpenData provider (ZIP/CSV parsing, station discovery).
