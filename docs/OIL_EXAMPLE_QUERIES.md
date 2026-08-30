# Oil Price Example SQL Queries

This document contains useful SQL queries for analyzing oil price data stored in the `oil_prices` table (PostgreSQL).

For provider details and data field descriptions, see [OIL_PROVIDERS.md](OIL_PROVIDERS.md).

## Get All Prices

```sql
SELECT provider, product_type, price_date, price_per_100l
FROM oil_prices
ORDER BY price_date DESC;
```

## Count Prices by Provider

See how much data each provider has collected.

```sql
SELECT
    provider,
    COUNT(*) AS total_prices,
    MIN(price_date) AS earliest_date,
    MAX(price_date) AS latest_date
FROM oil_prices
GROUP BY provider
ORDER BY total_prices DESC;
```

For a single provider:

```sql
SELECT COUNT(*)
FROM oil_prices
WHERE provider = 'heizoel24';
```

**Use cases:**
- Dashboard widgets showing data volume per provider
- Alerting when a provider has significantly fewer records than expected
- Comparing historical data availability between providers

## Get Latest Price

Retrieve the most recent price for a specific provider. Useful for displaying current prices on dashboards.

```sql
SELECT
    id,
    provider,
    product_type,
    price_date,
    price_per_100l,
    currency,
    scope,
    zip_code,
    fetched_at,
    created_at
FROM oil_prices
WHERE provider = 'heizoel24'
ORDER BY price_date DESC, created_at DESC
LIMIT 1;
```

**Use cases:**
- Display current heating oil prices on a dashboard
- Price comparison widgets
- Alerting when prices exceed a threshold
- Home automation integrations (e.g., notify when prices drop below target)

## Get Prices for a Date Range

Query all prices for a provider within a specific date range. Useful for generating reports and analyzing trends.

```sql
SELECT
    id,
    provider,
    product_type,
    price_date,
    price_per_100l,
    currency,
    scope,
    zip_code,
    fetched_at,
    created_at
FROM oil_prices
WHERE provider = 'heizoel24'
    AND price_date >= '2024-01-01'
    AND price_date <= '2024-12-31'
ORDER BY price_date ASC;
```

**Use cases:**
- Export historical data for external analysis
- Generate charts showing price trends over time
- Calculate average prices for a period
- Compare seasonal price variations
- Data backup and archival

## Price Trends

### Monthly Price Statistics

```sql
SELECT
    TO_CHAR(price_date, 'YYYY-MM') AS month,
    provider,
    ROUND(AVG(price_per_100l), 2) AS avg_price,
    MIN(price_per_100l) AS min_price,
    MAX(price_per_100l) AS max_price
FROM oil_prices
WHERE price_date >= CURRENT_DATE - INTERVAL '12 months'
GROUP BY TO_CHAR(price_date, 'YYYY-MM'), provider
ORDER BY month DESC, provider;
```

### Cheapest and Most Expensive Days

```sql
-- Cheapest days
SELECT price_date, price_per_100l
FROM oil_prices
WHERE provider = 'heizoel24'
    AND product_type = 'standard'
ORDER BY price_per_100l ASC
LIMIT 10;

-- Most expensive days
SELECT price_date, price_per_100l
FROM oil_prices
WHERE provider = 'heizoel24'
    AND product_type = 'standard'
ORDER BY price_per_100l DESC
LIMIT 10;
```

### Day-over-Day Change

Show the absolute and relative price movement compared to the previous recorded day.

```sql
WITH daily AS (
    SELECT
        price_date,
        price_per_100l,
        LAG(price_per_100l) OVER (ORDER BY price_date) AS previous_price
    FROM oil_prices
    WHERE provider = 'heizoel24'
        AND product_type = 'standard'
        AND price_date >= CURRENT_DATE - INTERVAL '90 days'
)
SELECT
    price_date,
    price_per_100l,
    previous_price,
    ROUND(price_per_100l - previous_price, 2) AS change_eur,
    ROUND(100.0 * (price_per_100l - previous_price) / NULLIF(previous_price, 0), 2) AS change_pct
FROM daily
WHERE previous_price IS NOT NULL
ORDER BY price_date DESC;
```

### Rolling Averages

Smooth out daily noise with 7-day and 30-day moving averages.

```sql
SELECT
    price_date,
    price_per_100l,
    ROUND(AVG(price_per_100l) OVER (
        ORDER BY price_date
        ROWS BETWEEN 6 PRECEDING AND CURRENT ROW
    ), 2) AS avg_7d,
    ROUND(AVG(price_per_100l) OVER (
        ORDER BY price_date
        ROWS BETWEEN 29 PRECEDING AND CURRENT ROW
    ), 2) AS avg_30d
FROM oil_prices
WHERE provider = 'heizoel24'
    AND product_type = 'standard'
    AND price_date >= CURRENT_DATE - INTERVAL '12 months'
ORDER BY price_date DESC;
```

## Product Types

### Discover Available Product Types

Product types are provider-specific. HeizOel24 stores a single `standard` product, while Hoyer derives its
values from the product names returned by the API, so query them instead of guessing.

```sql
SELECT
    provider,
    product_type,
    COUNT(*) AS records,
    MIN(price_date) AS first_seen,
    MAX(price_date) AS last_seen
FROM oil_prices
GROUP BY provider, product_type
ORDER BY provider, records DESC;
```

### Average Price per Product Type

```sql
SELECT
    product_type,
    COUNT(*) AS records,
    ROUND(AVG(price_per_100l), 2) AS avg_price,
    MIN(price_per_100l) AS min_price,
    MAX(price_per_100l) AS max_price
FROM oil_prices
WHERE provider = 'hoyer'
    AND price_date >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY product_type
ORDER BY avg_price;
```

## Regional Prices

### Prices by ZIP Code

Local providers store one price per ZIP code (`scope = 'local'`); national providers store a single
country-wide price with `zip_code` set to `NULL`.

```sql
SELECT
    zip_code,
    product_type,
    COUNT(*) AS records,
    ROUND(AVG(price_per_100l), 2) AS avg_price,
    MIN(price_per_100l) AS min_price,
    MAX(price_per_100l) AS max_price
FROM oil_prices
WHERE scope = 'local'
    AND price_date >= CURRENT_DATE - INTERVAL '90 days'
GROUP BY zip_code, product_type
ORDER BY zip_code, product_type;
```

### National vs. Local Comparison

Check whether local prices run above or below the national average.

```sql
SELECT
    price_date,
    ROUND(AVG(price_per_100l) FILTER (WHERE scope = 'national'), 2) AS national_avg,
    ROUND(AVG(price_per_100l) FILTER (WHERE scope = 'local'), 2) AS local_avg,
    ROUND(
        AVG(price_per_100l) FILTER (WHERE scope = 'local')
        - AVG(price_per_100l) FILTER (WHERE scope = 'national'), 2
    ) AS local_premium
FROM oil_prices
WHERE price_date >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY price_date
ORDER BY price_date DESC;
```

## Compare Providers

### Side-by-Side Daily Prices

```sql
SELECT
    price_date,
    MAX(CASE WHEN provider = 'heizoel24' THEN price_per_100l END) AS heizoel24,
    MAX(CASE WHEN provider = 'hoyer' THEN price_per_100l END) AS hoyer
FROM oil_prices
WHERE price_date >= CURRENT_DATE - INTERVAL '30 days'
    AND product_type = 'standard'
GROUP BY price_date
ORDER BY price_date DESC;
```

Note that `product_type = 'standard'` only matches HeizOel24. Replace it with one of Hoyer's own product
types (see [Discover Available Product Types](#discover-available-product-types)) to fill the `hoyer` column.

### Provider Deviation Analysis

Measure how much the providers disagree per month, across all their product types.

```sql
SELECT
    TO_CHAR(price_date, 'YYYY-MM') AS month,
    ROUND(AVG(price_per_100l) FILTER (WHERE provider = 'heizoel24'), 2) AS heizoel24_avg,
    ROUND(AVG(price_per_100l) FILTER (WHERE provider = 'hoyer'), 2) AS hoyer_avg,
    ROUND(
        AVG(price_per_100l) FILTER (WHERE provider = 'hoyer')
        - AVG(price_per_100l) FILTER (WHERE provider = 'heizoel24'), 2
    ) AS spread
FROM oil_prices
WHERE price_date >= CURRENT_DATE - INTERVAL '6 months'
GROUP BY TO_CHAR(price_date, 'YYYY-MM')
ORDER BY month DESC;
```

## Data Quality

### Data Completeness per Provider

`price_per_100l` is never NULL, so completeness is a question of how many days were actually collected
between the first and the last record.

```sql
SELECT
    provider,
    COUNT(*) AS total_rows,
    COUNT(DISTINCT price_date) AS distinct_days,
    COUNT(DISTINCT product_type) AS product_types,
    COUNT(DISTINCT zip_code) AS zip_codes,
    COUNT(raw_response) AS with_raw_response,
    ROUND(
        100.0 * COUNT(DISTINCT price_date)
        / NULLIF(MAX(price_date) - MIN(price_date) + 1, 0), 1
    ) AS day_coverage_pct
FROM oil_prices
GROUP BY provider
ORDER BY provider;
```

### Find Gaps in Daily Data

Detect missing dates where no price was recorded. Only HeizOel24 supports backfilling those gaps.

```sql
WITH date_series AS (
    SELECT generate_series(
        (SELECT MIN(price_date) FROM oil_prices WHERE provider = 'heizoel24'),
        (SELECT MAX(price_date) FROM oil_prices WHERE provider = 'heizoel24'),
        '1 day'::interval
    )::date AS expected_date
),
actual AS (
    SELECT DISTINCT price_date
    FROM oil_prices
    WHERE provider = 'heizoel24'
)
SELECT d.expected_date AS missing_date
FROM date_series d
LEFT JOIN actual a ON d.expected_date = a.price_date
WHERE a.price_date IS NULL
ORDER BY d.expected_date;
```

### Last Successful Fetch per Provider

Spot a scraper that stopped delivering data.

```sql
SELECT
    provider,
    MAX(fetched_at) AS last_fetch,
    MAX(price_date) AS latest_price_date,
    CURRENT_DATE - MAX(price_date) AS days_behind
FROM oil_prices
GROUP BY provider
ORDER BY last_fetch;
```

### Inspect the Raw API Response

Every row keeps the untouched API payload in `raw_response` (JSONB), which is handy when a value looks wrong.

```sql
SELECT
    provider,
    price_date,
    jsonb_pretty(raw_response) AS raw
FROM oil_prices
WHERE provider = 'hoyer'
    AND raw_response IS NOT NULL
ORDER BY price_date DESC
LIMIT 1;
```

## Buying Signals

### Best Day of Week to Buy

```sql
SELECT
    TO_CHAR(price_date, 'Day') AS day_of_week,
    ROUND(AVG(price_per_100l), 2) AS avg_price
FROM oil_prices
WHERE provider = 'heizoel24'
    AND price_date >= CURRENT_DATE - INTERVAL '6 months'
GROUP BY TO_CHAR(price_date, 'Day'), EXTRACT(DOW FROM price_date)
ORDER BY EXTRACT(DOW FROM price_date);
```

### Seasonal Averages by Month of Year

Aggregate across all available years to see which months are historically cheapest.

```sql
SELECT
    TO_CHAR(price_date, 'MM') AS month_of_year,
    COUNT(*) AS records,
    ROUND(AVG(price_per_100l), 2) AS avg_price,
    MIN(price_per_100l) AS min_price,
    MAX(price_per_100l) AS max_price
FROM oil_prices
WHERE provider = 'heizoel24'
    AND product_type = 'standard'
GROUP BY TO_CHAR(price_date, 'MM')
ORDER BY month_of_year;
```

### Current Price vs. 90-Day Low

```sql
WITH recent AS (
    SELECT price_date, price_per_100l
    FROM oil_prices
    WHERE provider = 'heizoel24'
        AND product_type = 'standard'
        AND price_date >= CURRENT_DATE - INTERVAL '90 days'
),
current_price AS (
    SELECT price_per_100l
    FROM recent
    ORDER BY price_date DESC
    LIMIT 1
)
SELECT
    (SELECT price_per_100l FROM current_price) AS current_price,
    MIN(price_per_100l) AS low_90d,
    MAX(price_per_100l) AS high_90d,
    ROUND(AVG(price_per_100l), 2) AS avg_90d,
    ROUND(
        100.0 * ((SELECT price_per_100l FROM current_price) - MIN(price_per_100l))
        / NULLIF(MIN(price_per_100l), 0), 1
    ) AS pct_above_low
FROM recent;
```

### Correlate Oil Prices with Weather

Join price data with weather observations to see how heating demand relates to price movement. The
counterpart query, starting from the weather side, lives in
[WEATHER_EXAMPLE_QUERIES.md](WEATHER_EXAMPLE_QUERIES.md).

```sql
SELECT
    o.price_date,
    o.price_per_100l AS oil_price_eur,
    ROUND(w.temperature_mean_c, 1) AS mean_temp_c,
    ROUND(GREATEST(15.0 - w.temperature_mean_c, 0), 1) AS hdd
FROM oil_prices o
JOIN weather_observations w
    ON o.price_date = w.observation_date
WHERE o.provider = 'heizoel24'
    AND o.product_type = 'standard'
    AND w.provider = 'openmeteo'
    AND o.price_date >= CURRENT_DATE - INTERVAL '6 months'
ORDER BY o.price_date DESC;
```
