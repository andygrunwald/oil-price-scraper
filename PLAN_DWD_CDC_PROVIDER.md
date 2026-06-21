# Plan: Implement DWD CDC-OpenData Weather Provider

## Context

The weather scraper already supports 4 providers (Open-Meteo, Bright Sky, Visual Crossing, OpenWeather). The DWD CDC-OpenData provider was deferred during the initial implementation due to its complexity (ZIP downloads, semicolon-delimited parsing, station discovery). This plan implements it as the 5th provider.

DWD CDC-OpenData provides **official German Weather Service station measurements** — quality-controlled daily climate data (daily KL) going back 50+ years. Unlike the other providers which are REST/JSON APIs, this one requires downloading ZIP files from an HTTP file server, extracting semicolon-delimited text files, and parsing them.

**No API key required. No rate limits. CC BY 4.0 license.**

---

## Milestone 1: Station Discovery (`stations.go`)

**Goal**: Download, parse, and search the DWD station list to find the nearest station to given coordinates.

**Create** `internal/weatherapi/dwdcdc/stations.go`

### Station list file

- **URL**: `https://opendata.dwd.de/climate_environment/CDC/observations_germany/climate/daily/kl/recent/KL_Tageswerte_Beschreibung_Stationen.txt`
- **Format**: Space-delimited text with a 2-line header (column names + dashes)
- **Columns** (fixed-width positions):
  1. `Stations_id` — 5-digit numeric ID (e.g., `00001`)
  2. `von_datum` — Start date (`YYYYMMDD`)
  3. `bis_datum` — End date (`YYYYMMDD`)
  4. `Stationshoehe` — Elevation in meters
  5. `geoBreite` — Latitude (decimal degrees)
  6. `geoLaenge` — Longitude (decimal degrees)
  7. `Stationsname` — Station name (German, may contain spaces)
  8. `Bundesland` — German state

- **Example lines**:
  ```
  00001 19370101 19860630   478   47.8413    8.8493 Aach                           Baden-Württemberg
  00003 18910101 20110331   202   50.7827    6.0941 Aachen                         Nordrhein-Westfalen
  ```

### Implementation

```go
type Station struct {
    ID        string    // 5-digit zero-padded ID
    Name      string
    Latitude  float64
    Longitude float64
    Elevation float64
    DateFrom  time.Time
    DateTo    time.Time
    State     string
}
```

**Functions**:
- `fetchStationList(ctx, client) ([]Station, error)` — Download and parse the station list file
- `parseStationList(reader io.Reader) ([]Station, error)` — Parse the space-delimited text, skip 2 header lines
- `findNearestStation(stations []Station, lat, lon float64) Station` — Haversine distance to find closest active station
- `haversineDistance(lat1, lon1, lat2, lon2 float64) float64` — Great-circle distance in km

**Parsing approach**: Fixed-width column positions. The file layout is:
- Chars 0-4: Station ID
- Chars 6-13: von_datum
- Chars 15-22: bis_datum
- Chars 24-28: Stationshoehe
- Chars 30-38: geoBreite
- Chars 40-49: geoLaenge
- Chars 51-90: Stationsname (may contain spaces — fixed-width is essential here)
- Chars 91+: Bundesland

Extract each field by substring, then `strings.TrimSpace()`. Positions must be verified against the actual file header line.

**Station for Duisburg** (~51.4556°N, 6.7623°E): The nearest station will be determined at runtime via Haversine lookup. Likely candidates include station 02110 (Duisburg-Baerl area) or nearby Essen/Mülheim stations.

### Verification
- Unit test: parse a fixture of 5-10 station lines, verify fields
- Unit test: Haversine distance calculation
- Unit test: nearest station selection

---

## Milestone 2: ZIP Download & Data Parsing (`parser.go`)

**Goal**: Download ZIP files, extract the data file, and parse semicolon-delimited daily records.

**Create** `internal/weatherapi/dwdcdc/parser.go`

### ZIP file URLs

- **Recent** (last ~500 days, daily updates, not fully QC'd):
  `https://opendata.dwd.de/climate_environment/CDC/observations_germany/climate/daily/kl/recent/tageswerte_KL_{STATION_ID}_akt.zip`

- **Historical** (quality-controlled, updated annually):
  `https://opendata.dwd.de/climate_environment/CDC/observations_germany/climate/daily/kl/historical/tageswerte_KL_{STATION_ID}_{FROM}_{TO}_hist.zip`

### Data file format inside ZIP

- **Filename pattern**: `produkt_klima_tag_{YYYYMMDD}_{YYYYMMDD}_{STATION_ID}.txt`
- **Format**: Semicolon-delimited (`;`) with header row
- **19 columns**:

| # | Column | Meaning | Unit | Mapping |
|---|--------|---------|------|---------|
| 1 | STATIONS_ID | Station ID | — | — |
| 2 | MESS_DATUM | Date | YYYYMMDD | `Date` |
| 3 | QN_3 | Quality wind | — | — |
| 4 | FX | Max wind gust | m/s | `WindGustMaxKmh` (×3.6) |
| 5 | FM | Mean wind speed | m/s | `WindSpeedMaxKmh` (×3.6) |
| 6 | QN_4 | Quality other | — | — |
| 7 | RSK | Precipitation | mm | `PrecipitationMmSum` |
| 8 | RSKF | Precip form | code | — |
| 9 | SDK | Sunshine | hours | `SunshineDurationS` (×3600) |
| 10 | SHK_TAG | Snow depth | cm | — |
| 11 | NM | Cloud cover | 1/8 | `CloudCoverPercent` (×12.5) |
| 12 | VPM | Vapor pressure | hPa | — |
| 13 | PM | Mean pressure | hPa | `PressureHpa` |
| 14 | TMK | Mean temperature | °C | `TemperatureMeanC` |
| 15 | UPM | Relative humidity | % | `HumidityPercent` |
| 16 | TXK | Max temperature | °C | `TemperatureMaxC` |
| 17 | TNK | Min temperature | °C | `TemperatureMinC` |
| 18 | TGK | Min ground temp | °C | — |
| 19 | eor | End of record | — | — |

- **Missing values**: `-999` or `-999.0` → `nil` (pointer stays nil)
- **Decimal separator**: dot (`.`)

### Implementation

```go
type dailyRecord struct {
    StationID string
    Date      time.Time
    FX        *float64 // max wind gust m/s
    FM        *float64 // mean wind speed m/s
    RSK       *float64 // precipitation mm
    SDK       *float64 // sunshine hours
    NM        *float64 // cloud cover 1/8
    PM        *float64 // pressure hPa
    TMK       *float64 // mean temp °C
    UPM       *float64 // humidity %
    TXK       *float64 // max temp °C
    TNK       *float64 // min temp °C
}
```

**Functions**:
- `downloadAndExtractZIP(ctx, client, url) ([]byte, error)` — Download ZIP into memory, find `produkt_klima_tag_*.txt`, return its contents
- `parseDataFile(data []byte) ([]dailyRecord, error)` — Parse semicolon-delimited lines, skip header, handle `-999`
- `filterByDateRange(records []dailyRecord, from, to time.Time) []dailyRecord` — Filter records to requested range
- `recordToWeatherResult(rec dailyRecord, lat, lon float64, rawBody []byte) models.WeatherResult` — Convert with unit conversions:
  - Wind: m/s → km/h (×3.6)
  - Sunshine: hours → seconds (×3600)
  - Cloud cover: okta (1/8) → percent (×12.5)
- `parseFloat(s string) *float64` — Parse float, return nil for `-999` / `-999.0` / empty

### Verification
- Unit test: parse fixture data file (5-10 lines from real DWD data)
- Unit test: `-999` handling → nil
- Unit test: unit conversions (m/s→km/h, hours→seconds, okta→percent)
- Unit test: date filtering

---

## Milestone 3: Main Provider (`dwdcdc.go`)

**Goal**: Wire station discovery and data parsing into the `weatherapi.Provider` interface.

**Create** `internal/weatherapi/dwdcdc/dwdcdc.go`

### Provider structure

```go
const providerName = "dwdcdc"

type Provider struct {
    logger     zerolog.Logger
    client     *http.Client
    stationID  string    // cached after first lookup
    stationLat float64   // station's actual coordinates
    stationLon float64
    mu         sync.Mutex
}
```

### Method implementations

- `Name()` → `"dwdcdc"`
- `SupportsBackfill()` → `true`
- `RequiresAPIKey()` → `false`
- `New(logger) *Provider` — constructor with 60s HTTP timeout (ZIP downloads can be large)

- `FetchCurrentWeather(ctx, lat, lon)`:
  1. Resolve station (call `ensureStation(ctx, lat, lon)`)
  2. Download **recent** ZIP for the station
  3. Parse data file
  4. Return the **most recent available day** (usually yesterday due to ~1 day lag). Log a debug message noting the actual date returned.
  5. Return as `[]models.WeatherResult`

- `FetchHistoricalWeather(ctx, lat, lon, from, to)`:
  1. Resolve station
  2. Download **historical** ZIP for the station
  3. If date range extends into recent period, also download **recent** ZIP
  4. Parse both, merge, deduplicate by date
  5. Filter to requested range
  6. Return as `[]models.WeatherResult`

- `ensureStation(ctx, lat, lon)` — Lazy station discovery:
  1. If `stationID` already set, return
  2. Fetch station list
  3. Find nearest station
  4. Cache `stationID`, `stationLat`, `stationLon`
  5. Log which station was selected (name, distance)

### Historical ZIP URL discovery

The historical ZIP filename includes the station's date range (e.g., `tageswerte_KL_02110_19210101_20231231_hist.zip`). Since we don't know the exact dates upfront, we need to:

**Option A**: Parse the station list for date ranges (von_datum/bis_datum) and construct the URL.
**Option B**: Fetch the directory listing and find the matching filename.

**Recommended**: Option A — use the station list's `DateFrom`/`DateTo` to construct the historical URL. This avoids an extra HTTP request.

### Verification
- Unit test: station caching (ensureStation called twice, only fetches once)
- Integration test: FetchCurrentWeather with a mock HTTP server serving fixture ZIP

---

## Milestone 4: Register Provider in CLI

**Goal**: Add `"dwdcdc"` to the provider switch statements in the weather scraper commands.

**Modify** `cmd/weatherscraper/cmd_run.go`, `cmd_scrape.go`, `cmd_backfill.go`:

Add case to each switch statement:
```go
case "dwdcdc":
    s.RegisterProvider(dwdcdc.New(logger))
```

No API key or extra constructor parameters needed.

### Verification
- `go build ./cmd/weatherscraper` compiles
- `weatherscraper scrape --help` shows dwdcdc as a valid option in provider docs

---

## Milestone 5: Tests

**Goal**: Comprehensive unit tests for all 3 new files.

### Test fixtures

**Create** `internal/weatherapi/dwdcdc/testdata/`:
- `stations.txt` — 5-10 real station lines from the DWD station list file
- `produkt_klima_tag_sample.txt` — 5-10 real data lines with header (including `-999` values)

### Test files

**Create** `internal/weatherapi/dwdcdc/stations_test.go`:
- `TestParseStationList` — Parse fixture, verify all fields
- `TestHaversineDistance` — Known distances (e.g., Berlin↔Munich ≈ 504km)
- `TestFindNearestStation` — Given Duisburg coords, find correct station from fixture

**Create** `internal/weatherapi/dwdcdc/parser_test.go`:
- `TestParseDataFile` — Parse fixture, verify all fields
- `TestMissingValueHandling` — `-999` → nil, `-999.0` → nil
- `TestUnitConversions` — m/s→km/h, hours→seconds, okta→percent
- `TestFilterByDateRange` — Include/exclude boundary dates
- `TestParseFloat` — Various inputs including empty strings and `-999`

**Create** `internal/weatherapi/dwdcdc/dwdcdc_test.go`:
- `TestProviderInterface` — Verify Name, SupportsBackfill, RequiresAPIKey

### Verification
- `go test ./internal/weatherapi/dwdcdc/...` — all tests pass
- `go test ./...` — full suite still passes

---

## Files Summary

### New files (6 files)
| File | Purpose |
|------|---------|
| `internal/weatherapi/dwdcdc/dwdcdc.go` | Main provider (interface impl, station caching, fetch orchestration) |
| `internal/weatherapi/dwdcdc/stations.go` | Station list download, parsing, nearest-station Haversine lookup |
| `internal/weatherapi/dwdcdc/parser.go` | ZIP download/extraction, semicolon-delimited parsing, unit conversions |
| `internal/weatherapi/dwdcdc/dwdcdc_test.go` | Provider interface tests |
| `internal/weatherapi/dwdcdc/stations_test.go` | Station parsing + Haversine tests |
| `internal/weatherapi/dwdcdc/parser_test.go` | Data parsing, missing values, unit conversion tests |

### Modified files (3 files)
| File | Change |
|------|--------|
| `cmd/weatherscraper/cmd_run.go` | Add `"dwdcdc"` case to provider switch |
| `cmd/weatherscraper/cmd_scrape.go` | Add `"dwdcdc"` case to provider switch |
| `cmd/weatherscraper/cmd_backfill.go` | Add `"dwdcdc"` case to provider switch |

### Test fixture files (2 files)
| File | Purpose |
|------|---------|
| `internal/weatherapi/dwdcdc/testdata/stations.txt` | Sample station list lines |
| `internal/weatherapi/dwdcdc/testdata/produkt_klima_tag_sample.txt` | Sample daily data with header |

---

## Implementation Sequence

```
M1 (stations.go) → M2 (parser.go) → M3 (dwdcdc.go) → M4 (CLI registration) → M5 (tests)
```

All milestones are sequential — each builds on the previous. Tests in M5 can also be written alongside each milestone.
