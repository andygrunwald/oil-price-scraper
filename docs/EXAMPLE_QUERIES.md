# Example SQL Queries

This document contains useful SQL queries for analyzing oil price data stored in the `oil_prices` table.

## Count Prices by Provider

Get the total number of price records for a specific provider. Useful for monitoring data collection and comparing provider coverage.

```sql
SELECT COUNT(*)
FROM `oil_prices`
WHERE `provider` = 'heizoel24';
```

**Use cases:**
- Dashboard widgets showing data volume per provider
- Alerting when a provider has significantly fewer records than expected
- Comparing historical data availability between providers

## Get Latest Price

Retrieve the most recent price for a specific provider. Useful for displaying current prices on dashboards.

```sql
SELECT
    `id`,
    `provider`,
    `product_type`,
    `price_date`,
    `price_per_100l`,
    `currency`,
    `scope`,
    `zip_code`,
    `fetched_at`,
    `created_at`
FROM `oil_prices`
WHERE `provider` = 'heizoel24'
ORDER BY `price_date` DESC, `created_at` DESC
LIMIT 1;
```

**Use cases:**
- Display current heating oil prices on a dashboard
- Price comparison widgets
- Alerting when prices exceed a threshold
- Home automation integrations (e.g., notify when prices drop below target)

## Get Prices for Date Range

Query all prices for a provider within a specific date range. Useful for generating reports and analyzing trends.

```sql
SELECT
    `id`,
    `provider`,
    `product_type`,
    `price_date`,
    `price_per_100l`,
    `currency`,
    `scope`,
    `zip_code`,
    `fetched_at`,
    `created_at`
FROM `oil_prices`
WHERE `provider` = 'heizoel24'
    AND `price_date` >= '2024-01-01'
    AND `price_date` <= '2024-12-31'
ORDER BY `price_date` ASC;
```

**Use cases:**
- Export historical data for external analysis
- Generate charts showing price trends over time
- Calculate average prices for a period
- Compare seasonal price variations
- Data backup and archival

## Additional Useful Queries

### Price Statistics by Month

```sql
SELECT
    DATE_FORMAT(`price_date`, '%Y-%m') AS `month`,
    `provider`,
    AVG(`price_per_100l`) AS `avg_price`,
    MIN(`price_per_100l`) AS `min_price`,
    MAX(`price_per_100l`) AS `max_price`
FROM `oil_prices`
WHERE `price_date` >= DATE_SUB(CURDATE(), INTERVAL 12 MONTH)
GROUP BY DATE_FORMAT(`price_date`, '%Y-%m'), `provider`
ORDER BY `month` DESC, `provider`;
```

### Find Best Price Day of Week

```sql
SELECT
    DAYNAME(`price_date`) AS `day_of_week`,
    AVG(`price_per_100l`) AS `avg_price`
FROM `oil_prices`
WHERE `provider` = 'heizoel24'
    AND `price_date` >= DATE_SUB(CURDATE(), INTERVAL 6 MONTH)
GROUP BY DAYNAME(`price_date`), DAYOFWEEK(`price_date`)
ORDER BY DAYOFWEEK(`price_date`);
```

### Compare Providers

```sql
SELECT
    `price_date`,
    MAX(CASE WHEN `provider` = 'heizoel24' THEN `price_per_100l` END) AS `heizoel24`,
    MAX(CASE WHEN `provider` = 'hoyer' THEN `price_per_100l` END) AS `hoyer`
FROM `oil_prices`
WHERE `price_date` >= DATE_SUB(CURDATE(), INTERVAL 30 DAY)
    AND `product_type` = 'standard'
GROUP BY `price_date`
ORDER BY `price_date` DESC;
```
