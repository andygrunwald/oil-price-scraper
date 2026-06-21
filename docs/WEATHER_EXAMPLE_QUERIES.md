# Weather Example SQL Queries

This document contains useful SQL queries for analyzing weather data stored in the `weather_observations` table (PostgreSQL).

For provider details and data field descriptions, see [WEATHER_PROVIDERS.md](WEATHER_PROVIDERS.md).

## Get All Observations

```sql
SELECT provider, observation_date, temperature_mean_c, precipitation_mm_sum
FROM weather_observations
ORDER BY observation_date DESC;
```

## Count Observations by Provider

See how much data each provider has collected.

```sql
SELECT
    provider,
    COUNT(*) AS total_observations,
    MIN(observation_date) AS earliest_date,
    MAX(observation_date) AS latest_date
FROM weather_observations
GROUP BY provider
ORDER BY total_observations DESC;
```

**Use cases:**
- Monitor data collection health per provider
- Compare historical coverage across providers
- Detect gaps in data collection

## Get Latest Observation

Retrieve the most recent weather observation from a specific provider.

```sql
SELECT *
FROM weather_observations
WHERE provider = 'openmeteo'
ORDER BY observation_date DESC
LIMIT 1;
```

## Get Observations for a Date Range

```sql
SELECT
    observation_date,
    temperature_min_c,
    temperature_max_c,
    temperature_mean_c,
    precipitation_mm_sum,
    sunshine_duration_s / 3600.0 AS sunshine_hours
FROM weather_observations
WHERE provider = 'openmeteo'
    AND observation_date >= '2024-01-01'
    AND observation_date <= '2024-12-31'
ORDER BY observation_date ASC;
```

## Temperature Analysis

### Monthly Temperature Statistics

Average, minimum, and maximum temperatures by month — useful for identifying seasonal patterns.

```sql
SELECT
    TO_CHAR(observation_date, 'YYYY-MM') AS month,
    provider,
    ROUND(AVG(temperature_mean_c), 1) AS avg_temp,
    ROUND(MIN(temperature_min_c), 1) AS coldest_min,
    ROUND(MAX(temperature_max_c), 1) AS warmest_max,
    ROUND(AVG(temperature_max_c - temperature_min_c), 1) AS avg_daily_range
FROM weather_observations
WHERE observation_date >= CURRENT_DATE - INTERVAL '12 months'
GROUP BY TO_CHAR(observation_date, 'YYYY-MM'), provider
ORDER BY month DESC, provider;
```

### Hottest and Coldest Days

Find extreme temperature days in the dataset.

```sql
-- Hottest days
SELECT provider, observation_date, temperature_max_c
FROM weather_observations
WHERE temperature_max_c IS NOT NULL
ORDER BY temperature_max_c DESC
LIMIT 10;

-- Coldest days
SELECT provider, observation_date, temperature_min_c
FROM weather_observations
WHERE temperature_min_c IS NOT NULL
ORDER BY temperature_min_c ASC
LIMIT 10;
```

### Frost Days per Month

Count days where the minimum temperature dropped below 0°C.

```sql
SELECT
    TO_CHAR(observation_date, 'YYYY-MM') AS month,
    COUNT(*) AS frost_days
FROM weather_observations
WHERE provider = 'openmeteo'
    AND temperature_min_c < 0
GROUP BY TO_CHAR(observation_date, 'YYYY-MM')
ORDER BY month;
```

### Summer Days and Hot Days

Count days exceeding temperature thresholds (German meteorological definitions).

```sql
SELECT
    EXTRACT(YEAR FROM observation_date) AS year,
    COUNT(*) FILTER (WHERE temperature_max_c >= 25) AS summer_days,    -- Sommertage (≥25°C)
    COUNT(*) FILTER (WHERE temperature_max_c >= 30) AS hot_days,       -- Heisse Tage (≥30°C)
    COUNT(*) FILTER (WHERE temperature_max_c >= 35) AS desert_days,    -- Wüstentage (≥35°C)
    COUNT(*) FILTER (WHERE temperature_min_c >= 20) AS tropical_nights -- Tropennächte (≥20°C)
FROM weather_observations
WHERE provider = 'openmeteo'
GROUP BY EXTRACT(YEAR FROM observation_date)
ORDER BY year;
```

## Precipitation Analysis

### Monthly Precipitation Totals

```sql
SELECT
    TO_CHAR(observation_date, 'YYYY-MM') AS month,
    provider,
    ROUND(SUM(precipitation_mm_sum), 1) AS total_precipitation_mm,
    COUNT(*) FILTER (WHERE precipitation_mm_sum > 0) AS rainy_days,
    COUNT(*) FILTER (WHERE precipitation_mm_sum = 0) AS dry_days,
    ROUND(MAX(precipitation_mm_sum), 1) AS max_daily_precipitation_mm
FROM weather_observations
WHERE observation_date >= CURRENT_DATE - INTERVAL '12 months'
    AND precipitation_mm_sum IS NOT NULL
GROUP BY TO_CHAR(observation_date, 'YYYY-MM'), provider
ORDER BY month DESC, provider;
```

### Longest Dry Streak

Find the longest consecutive period without precipitation.

```sql
WITH daily AS (
    SELECT
        observation_date,
        precipitation_mm_sum,
        CASE WHEN precipitation_mm_sum = 0 THEN 0 ELSE 1 END AS has_rain
    FROM weather_observations
    WHERE provider = 'openmeteo'
        AND precipitation_mm_sum IS NOT NULL
),
groups AS (
    SELECT
        observation_date,
        has_rain,
        SUM(has_rain) OVER (ORDER BY observation_date) AS grp
    FROM daily
),
streaks AS (
    SELECT
        MIN(observation_date) AS streak_start,
        MAX(observation_date) AS streak_end,
        COUNT(*) AS streak_days
    FROM groups
    WHERE has_rain = 0
    GROUP BY grp
)
SELECT streak_start, streak_end, streak_days
FROM streaks
ORDER BY streak_days DESC
LIMIT 5;
```

### Wettest Days

```sql
SELECT provider, observation_date, precipitation_mm_sum
FROM weather_observations
WHERE precipitation_mm_sum IS NOT NULL
ORDER BY precipitation_mm_sum DESC
LIMIT 10;
```

## Sunshine and Cloud Analysis

### Monthly Sunshine Hours

```sql
SELECT
    TO_CHAR(observation_date, 'YYYY-MM') AS month,
    provider,
    ROUND(SUM(sunshine_duration_s) / 3600.0, 1) AS total_sunshine_hours,
    ROUND(AVG(sunshine_duration_s) / 3600.0, 1) AS avg_daily_sunshine_hours
FROM weather_observations
WHERE sunshine_duration_s IS NOT NULL
    AND observation_date >= CURRENT_DATE - INTERVAL '12 months'
GROUP BY TO_CHAR(observation_date, 'YYYY-MM'), provider
ORDER BY month DESC, provider;
```

### Overcast vs. Clear Days

```sql
SELECT
    EXTRACT(YEAR FROM observation_date) AS year,
    EXTRACT(MONTH FROM observation_date) AS month,
    COUNT(*) FILTER (WHERE cloud_cover_percent < 20) AS clear_days,
    COUNT(*) FILTER (WHERE cloud_cover_percent BETWEEN 20 AND 80) AS partly_cloudy_days,
    COUNT(*) FILTER (WHERE cloud_cover_percent > 80) AS overcast_days
FROM weather_observations
WHERE provider = 'openmeteo'
    AND cloud_cover_percent IS NOT NULL
GROUP BY EXTRACT(YEAR FROM observation_date), EXTRACT(MONTH FROM observation_date)
ORDER BY year, month;
```

## Wind Analysis

### Windiest Days

```sql
SELECT provider, observation_date, wind_speed_max_kmh, wind_gust_max_kmh
FROM weather_observations
WHERE wind_speed_max_kmh IS NOT NULL
ORDER BY wind_speed_max_kmh DESC
LIMIT 10;
```

### Storm Days

Count days with wind gusts exceeding Beaufort thresholds.

```sql
SELECT
    EXTRACT(YEAR FROM observation_date) AS year,
    COUNT(*) FILTER (WHERE wind_gust_max_kmh >= 62) AS storm_days,           -- Beaufort 8+
    COUNT(*) FILTER (WHERE wind_gust_max_kmh >= 75) AS severe_storm_days,    -- Beaufort 9+
    COUNT(*) FILTER (WHERE wind_gust_max_kmh >= 103) AS hurricane_force_days -- Beaufort 12
FROM weather_observations
WHERE provider = 'openmeteo'
    AND wind_gust_max_kmh IS NOT NULL
GROUP BY EXTRACT(YEAR FROM observation_date)
ORDER BY year;
```

## Compare Providers

### Side-by-Side Temperature Comparison

Compare daily mean temperatures from different providers to check consistency.

```sql
SELECT
    o.observation_date,
    ROUND(o.temperature_mean_c, 1) AS openmeteo,
    ROUND(b.temperature_mean_c, 1) AS brightsky,
    ROUND(d.temperature_mean_c, 1) AS dwdcdc,
    ROUND(ABS(o.temperature_mean_c - b.temperature_mean_c), 1) AS diff_om_bs
FROM weather_observations o
JOIN weather_observations b
    ON o.observation_date = b.observation_date
    AND o.latitude = b.latitude AND o.longitude = b.longitude
JOIN weather_observations d
    ON o.observation_date = d.observation_date
    AND o.latitude = d.latitude AND o.longitude = d.longitude
WHERE o.provider = 'openmeteo'
    AND b.provider = 'brightsky'
    AND d.provider = 'dwdcdc'
    AND o.observation_date >= CURRENT_DATE - INTERVAL '30 days'
ORDER BY o.observation_date DESC;
```

### Provider Deviation Analysis

Measure how much providers disagree on mean temperature.

```sql
SELECT
    TO_CHAR(observation_date, 'YYYY-MM') AS month,
    ROUND(AVG(temperature_mean_c) FILTER (WHERE provider = 'openmeteo'), 1) AS openmeteo_avg,
    ROUND(AVG(temperature_mean_c) FILTER (WHERE provider = 'brightsky'), 1) AS brightsky_avg,
    ROUND(AVG(temperature_mean_c) FILTER (WHERE provider = 'dwdcdc'), 1) AS dwdcdc_avg,
    ROUND(
        MAX(AVG(temperature_mean_c)) OVER () -
        MIN(AVG(temperature_mean_c)) OVER (), 1
    ) AS max_spread
FROM weather_observations
WHERE temperature_mean_c IS NOT NULL
    AND observation_date >= CURRENT_DATE - INTERVAL '6 months'
GROUP BY TO_CHAR(observation_date, 'YYYY-MM')
ORDER BY month DESC;
```

## Data Quality

### Find Days with Missing Data

```sql
SELECT
    provider,
    observation_date,
    CASE WHEN temperature_mean_c IS NULL THEN 'temp' ELSE NULL END AS missing_temp,
    CASE WHEN precipitation_mm_sum IS NULL THEN 'precip' ELSE NULL END AS missing_precip,
    CASE WHEN wind_speed_max_kmh IS NULL THEN 'wind' ELSE NULL END AS missing_wind,
    CASE WHEN sunshine_duration_s IS NULL THEN 'sunshine' ELSE NULL END AS missing_sunshine
FROM weather_observations
WHERE temperature_mean_c IS NULL
    OR precipitation_mm_sum IS NULL
    OR wind_speed_max_kmh IS NULL
ORDER BY observation_date DESC, provider;
```

### Data Completeness per Provider

```sql
SELECT
    provider,
    COUNT(*) AS total_days,
    COUNT(temperature_mean_c) AS has_temp,
    COUNT(precipitation_mm_sum) AS has_precip,
    COUNT(wind_speed_max_kmh) AS has_wind,
    COUNT(wind_gust_max_kmh) AS has_gust,
    COUNT(sunshine_duration_s) AS has_sunshine,
    COUNT(cloud_cover_percent) AS has_cloud,
    COUNT(humidity_percent) AS has_humidity,
    COUNT(pressure_hpa) AS has_pressure,
    ROUND(100.0 * COUNT(sunshine_duration_s) / COUNT(*), 1) AS sunshine_pct
FROM weather_observations
GROUP BY provider
ORDER BY provider;
```

### Find Gaps in Daily Data

Detect missing dates where no observation was recorded.

```sql
WITH date_series AS (
    SELECT generate_series(
        (SELECT MIN(observation_date) FROM weather_observations WHERE provider = 'openmeteo'),
        (SELECT MAX(observation_date) FROM weather_observations WHERE provider = 'openmeteo'),
        '1 day'::interval
    )::date AS expected_date
),
actual AS (
    SELECT DISTINCT observation_date
    FROM weather_observations
    WHERE provider = 'openmeteo'
)
SELECT d.expected_date AS missing_date
FROM date_series d
LEFT JOIN actual a ON d.expected_date = a.observation_date
WHERE a.observation_date IS NULL
ORDER BY d.expected_date;
```

## Heating and Energy

### Heating Degree Days (HDD)

Calculate heating degree days with a base temperature of 15°C — useful for estimating heating energy demand (and correlating with oil prices).

```sql
SELECT
    TO_CHAR(observation_date, 'YYYY-MM') AS month,
    ROUND(SUM(GREATEST(15.0 - temperature_mean_c, 0)), 1) AS heating_degree_days
FROM weather_observations
WHERE provider = 'openmeteo'
    AND temperature_mean_c IS NOT NULL
    AND observation_date >= CURRENT_DATE - INTERVAL '12 months'
GROUP BY TO_CHAR(observation_date, 'YYYY-MM')
ORDER BY month;
```

### Cooling Degree Days (CDD)

```sql
SELECT
    TO_CHAR(observation_date, 'YYYY-MM') AS month,
    ROUND(SUM(GREATEST(temperature_mean_c - 18.0, 0)), 1) AS cooling_degree_days
FROM weather_observations
WHERE provider = 'openmeteo'
    AND temperature_mean_c IS NOT NULL
    AND observation_date >= CURRENT_DATE - INTERVAL '12 months'
GROUP BY TO_CHAR(observation_date, 'YYYY-MM')
ORDER BY month;
```

### Correlate Weather with Oil Prices

Join weather and oil price data to analyze the relationship between temperature and heating oil prices.

```sql
SELECT
    w.observation_date,
    ROUND(w.temperature_mean_c, 1) AS mean_temp_c,
    ROUND(w.precipitation_mm_sum, 1) AS precip_mm,
    o.price_per_100l AS oil_price_eur,
    ROUND(GREATEST(15.0 - w.temperature_mean_c, 0), 1) AS hdd
FROM weather_observations w
JOIN oil_prices o
    ON w.observation_date = o.price_date
WHERE w.provider = 'openmeteo'
    AND o.provider = 'heizoel24'
    AND o.product_type = 'standard'
    AND w.observation_date >= CURRENT_DATE - INTERVAL '6 months'
ORDER BY w.observation_date DESC;
```
