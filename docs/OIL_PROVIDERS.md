# Oil Price Providers

This document describes each oil price provider integrated into the oil scraper, including API details, data coverage, authentication requirements, and a detailed breakdown of which data points are scraped and stored versus available but not captured.

All providers return daily oil price observations that are stored in the `oil_prices` PostgreSQL table. See [EXAMPLE_QUERIES.md](EXAMPLE_QUERIES.md) for SQL query examples.

## Table of Contents

- [Data Points Stored](#data-points-stored)
- [Provider Comparison](#provider-comparison)
- [HeizOel24](#heizoel24)
- [Hoyer](#hoyer)
- [General Limitations](#general-limitations)

---

## Data Points Stored

Every provider maps its response into the same unified schema. The following columns are stored in the `oil_prices` table for each price record:

| Column | Type | Unit | Description |
|--------|------|------|-------------|
| `provider` | VARCHAR(50) | — | Provider identifier (`heizoel24`, `hoyer`) |
| `product_type` | VARCHAR(50) | — | Product variant (e.g., `standard`, `premiumheizoel`) |
| `price_date` | DATE | — | Date the price is valid for |
| `price_per_100l` | DECIMAL(10,4) | EUR | Price per 100 liters (gross, including taxes) |
| `currency` | VARCHAR(10) | — | Currency code (always `EUR`) |
| `scope` | VARCHAR(10) | — | `national` (nationwide average) or `local` (zip-code specific) |
| `zip_code` | VARCHAR(10) | — | Zip code for local prices (`NULL` for national) |
| `raw_response` | JSONB | — | Original API response (optional, controlled by `--store-raw-response`) |
| `fetched_at` | TIMESTAMP | — | When the API was called |

The unique constraint `(provider, product_type, price_date, zip_code)` prevents duplicate entries.

---

## Provider Comparison

| Feature | HeizOel24 | Hoyer |
|---------|:---------:|:-----:|
| Price Scope | National (Germany-wide average) | Local (zip-code specific) |
| Product Types | 1 (`standard`) | Multiple (varies by region) |
| Backfill Support | ✅ (date range queries) | ❌ (current prices only) |
| API Key Required | No | No |
| API Documentation | None (unofficial) | None (unofficial) |
| Rate Limits | Unknown | Unknown |
| Price Field | Gross (incl. taxes) | Gross (incl. taxes) |
| Response Format | JSON | JSON |

---

## HeizOel24

- **Website**: [https://www.heizoel24.de](https://www.heizoel24.de)
- **Provider name**: `heizoel24`
- **API Key**: Not required
- **API Documentation**: None — this is an unofficial, undocumented API reverse-engineered from the HeizOel24 website. There are no public docs, no versioning guarantees, and no SLA.
- **Rate Limits**: Unknown (no documentation available)

### Endpoint

| Purpose | URL |
|---------|-----|
| Price History | `https://www.heizoel24.de/api/chartapi/GetAveragePriceHistory` |

**Request parameters** (GET query string):

| Parameter | Value | Description |
|-----------|-------|-------------|
| `countryId` | `1` | Germany |
| `minDate` | `YYYY-MM-DD` | Start date |
| `maxDate` | `YYYY-MM-DD` | End date |

**Example request**:
```
GET https://www.heizoel24.de/api/chartapi/GetAveragePriceHistory?countryId=1&minDate=2024-01-01&maxDate=2024-01-31
```

### Data Points

**API response fields:**

| API Field | Stored As | Unit Conversion | Notes |
|-----------|-----------|-----------------|-------|
| `Values[].Date` | `price_date` | Millisecond Unix timestamp → `time.Time` | Divided by 1000, converted to UTC |
| `Values[].Value` | `price_per_100l` | None (already EUR/100L) | Direct mapping |

**Hardcoded values** (not from API):

| Field | Value | Notes |
|-------|-------|-------|
| `currency` | `EUR` | Always Euro |
| `product_type` | `standard` | HeizOel24 only reports one product type |
| `scope` | `national` | Nationwide average price |
| `zip_code` | `NULL` | Not applicable for national scope |

### API Response Fields Available But Not Stored

| API Field | Type | Description | Why Not Stored |
|-----------|------|-------------|----------------|
| `Currency` | string | Currency identifier | Hardcoded to EUR |
| `ProductName` | string | Product name | Only one product type ("standard") |
| `ChartName` | string | Chart display name | Display metadata |
| `ChartUnit` | string | Unit for chart display | Display metadata |
| `CurrentPrice` | float64 | Current price snapshot | Redundant with latest `Values[]` entry |
| `ChangePercent` | float64 | Price change percentage | Derived metric, can be computed from stored data |

### How FetchCurrentPrices Works

Calls `FetchHistoricalPrices` with a 2-day window (yesterday to today) to capture the most recent price point.

### Known Limitations

- **Unofficial API**: This endpoint is not publicly documented by HeizOel24. It could change or disappear without notice. There is no versioning, no changelog, and no deprecation policy.
- **National average only**: Returns a single nationwide average price for Germany. Does not support regional or zip-code-specific queries.
- **Single product type**: Only reports "standard" heating oil (Standardheizöl). Premium, eco, or other product variants are not available.
- **No retry logic**: A single HTTP request is made per scrape. Network errors, timeouts, or server errors cause immediate failure with no automatic retry.
- **No rate limit awareness**: Since rate limits are unknown, there is no backoff or throttling mechanism. Aggressive scraping could potentially lead to IP blocking.
- **Timestamp precision**: Prices are stored with date-only precision. Intraday price changes (if any) are not captured.
- **Price type unclear**: The API does not explicitly document whether the price includes all taxes and fees. Based on the website, it appears to be a gross consumer price.
- **Data freshness**: It is unknown how frequently HeizOel24 updates their average price data internally. There may be a delay between market price changes and API updates.
- **30-second HTTP timeout**: Large historical date ranges may time out.
- **User-Agent rotation**: The scraper rotates browser User-Agent strings to avoid bot detection, which suggests the API may have anti-scraping measures.

---

## Hoyer

- **Website**: [https://www.hoyer.de](https://www.hoyer.de)
- **Heating oil page**: [https://www.hoyer.de/heizoel/](https://www.hoyer.de/heizoel/)
- **Provider name**: `hoyer`
- **API Key**: Not required
- **API Documentation**: None — this is an unofficial, undocumented API reverse-engineered from the Hoyer website/app. There are no public docs, no versioning guarantees, and no SLA.
- **Rate Limits**: Unknown (no documentation available)

### Endpoint

| Purpose | URL Pattern |
|---------|-------------|
| Current Prices | `https://api.hoyer.de/rest/heatingoil/{ZIP_CODE}/{ORDER_AMOUNT}/1` |

**URL path parameters**:

| Parameter | Example | Description |
|-----------|---------|-------------|
| `ZIP_CODE` | `47259` | German postal code for delivery |
| `ORDER_AMOUNT` | `3000` | Order quantity in liters |
| `1` | `1` | Number of delivery locations (hardcoded) |

**Example request**:
```
GET https://api.hoyer.de/rest/heatingoil/47259/3000/1
```

### Data Points

**Per-product mapping** (the API returns multiple products per request):

| API Field | Stored As | Unit Conversion | Notes |
|-----------|-----------|-----------------|-------|
| `Products[].Prices.PriceGross` | `price_per_100l` | German format → float (`"90,99"` → `90.99`) | Gross price including taxes |
| `Products[].Name` | `product_type` | Normalized (see below) | Lowercase, hyphens, umlaut replacement |

**Hardcoded/derived values:**

| Field | Value | Notes |
|-------|-------|-------|
| `currency` | `EUR` | Always Euro |
| `scope` | `local` | Zip-code-specific pricing |
| `zip_code` | From `--zip-code` flag | The configured zip code |
| `price_date` | Today (truncated to 00:00 UTC) | No historical date from API |

### Product Name Normalization

Hoyer returns German product names with umlauts and spaces. These are normalized:

1. Convert to lowercase
2. Replace spaces with hyphens
3. Replace German characters: `ö` → `oe`, `ä` → `ae`, `ü` → `ue`, `ß` → `ss`

**Examples**: `"Standardheizöl"` → `"standardheizoeel"`, `"Premiumheizöl"` → `"premiumheizoeel"`

### API Response Fields Available But Not Stored

| API Field | Type | Description | Why Not Stored |
|-----------|------|-------------|----------------|
| `Products[].ID` | int | Product identifier | Internal ID, not meaningful |
| `Products[].Description` | []string | Product description lines | Marketing text |
| `Products[].BasePrice` | float64 | Base/net price | We store PriceGross (gross with taxes) |
| `Products[].Prices.PriceNet` | string | Net price (German format) | We store PriceGross |
| `Products[].Prices.Taxes` | string | Tax amount | Can be derived from gross - net |
| `Products[].Prices.PriceTotalNet` | string | Total net price for order | Order-specific, not per-100L |
| `Products[].Prices.PriceTotalGross` | string | Total gross price for order | Order-specific, not per-100L |
| `Products[].Prices.TaxesTotal` | string | Total taxes for order | Order-specific |
| `Products[].Prices.WithAction` | *string | Promotional price | Promotional/temporary pricing |
| `Products[].Prices.TotalWithAction` | *string | Total with promotion | Promotional/temporary pricing |
| `Products[].Prices.PriceActionDifference` | float64 | Promotion discount | Promotional/temporary pricing |
| `Products[].IsPremium` | bool | Premium product flag | Product metadata |
| `Products[].IsClimateNeutral` | bool | Climate-neutral flag | Product metadata |
| `Products[].Days` | int | Delivery timeframe in days | Logistics info |
| `Products[].DeliveryTimeType` | string | Delivery time category | Logistics info |
| `Settings` | object | Settings object | Unused/empty in practice |

### Known Limitations

- **Unofficial API**: This endpoint is not publicly documented by Hoyer. It could change or disappear without notice. There is no versioning, no changelog, and no deprecation policy.
- **No historical data**: Hoyer only provides current prices. There is no way to query past prices. `FetchHistoricalPrices` returns an error. Backfill is not supported.
- **Local pricing only**: Prices are specific to a zip code and order amount. There is no national average.
- **Order amount affects price**: The price per 100L varies depending on the configured order amount (e.g., 3000L vs 1000L). Changing the `--order-amount` flag changes the price returned, which may create inconsistencies in stored data.
- **German decimal format**: Prices are returned as German-formatted strings (comma as decimal separator, e.g., `"90,99"`). Parsing errors cause the product to be silently skipped with a warning log.
- **No retry logic**: A single HTTP request is made per scrape. Network errors, timeouts, or server errors cause immediate failure with no automatic retry.
- **No rate limit awareness**: Since rate limits are unknown, there is no backoff or throttling mechanism.
- **Product availability varies**: Different zip codes may return different product sets. The product types stored depend on what Hoyer offers in that region.
- **Price type**: Uses `PriceGross` (gross price including taxes), not `BasePrice` or `PriceNet`. This was explicitly changed in commit `fc0fe90` to match consumer-facing prices.
- **Date precision**: Price date is always set to today (00:00 UTC). If the scraper runs multiple times per day, the upsert logic will overwrite the earlier price.
- **30-second HTTP timeout**: May be insufficient for slow API responses.
- **User-Agent rotation**: The scraper rotates browser User-Agent strings to avoid bot detection.
- **Promotional prices not captured**: The API returns promotional/action prices (`WithAction`, `TotalWithAction`) but these are not stored.

---

## General Limitations

These limitations apply to the oil scraper system as a whole, regardless of provider:

### No Retry Logic
Both providers make a single HTTP request per scrape attempt. If the request fails (network error, timeout, HTTP error), the failure is logged but not retried. The next scrape attempt occurs at the next scheduled time (default: daily at 06:00).

### Sequential Provider Execution
Providers are scraped one after another, not in parallel. If one provider is slow or fails, it delays scraping of subsequent providers.

### No Circuit Breaker
There is no circuit breaker pattern. If a provider API is consistently failing, the scraper will continue attempting requests at every scheduled interval without any backoff.

### No Request Caching
Each scrape makes fresh HTTP requests. There is no caching layer to reduce API load or provide fallback data during outages.

### No Data Validation
Prices returned by the API are stored as-is without sanity checks. There is no validation for reasonable price ranges (e.g., detecting if a price is 10× higher than expected due to an API error).

### Duplicate Detection
The scraper checks for existing records before inserting, and the database has a unique constraint as a safety net. However, the `ON CONFLICT DO UPDATE` behavior means a re-scrape will overwrite the previous record's `price_per_100l`, `raw_response`, and `fetched_at` — potentially losing the original data point if the API returns a corrected value later.

### Raw Response Storage
Raw API responses are only stored if `--store-raw-response` is enabled (default: `false`). Without this, there is no way to audit or re-parse historical API responses.

### Backfill Delay Parameters Unused
The `--min-delay` and `--max-delay` flags in the backfill command are accepted but never actually used in the implementation. HeizOel24 fetches the entire date range in a single API call, so inter-request delays are not needed.
