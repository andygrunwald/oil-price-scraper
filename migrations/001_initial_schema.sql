-- Oil Price Scraper - Initial Schema
-- Creates the database and oil_prices table

CREATE DATABASE IF NOT EXISTS `oilprices`;
USE `oilprices`;

CREATE TABLE IF NOT EXISTS `oil_prices` (
    `id`              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `provider`        VARCHAR(50) NOT NULL COMMENT 'Source API (e.g., heizoel24, hoyer)',
    `product_type`    VARCHAR(50) NOT NULL DEFAULT 'standard' COMMENT 'Product variant (e.g., standard, bestpreis, eco)',
    `price_date`      DATE NOT NULL COMMENT 'Date the price is valid for',
    `price_per_100l`  DECIMAL(10, 4) NOT NULL COMMENT 'Price in EUR per 100 liters',
    `currency`        VARCHAR(10) NOT NULL DEFAULT 'EUR' COMMENT 'Currency code',
    `scope`           ENUM('local', 'national') NOT NULL COMMENT 'Geographical scope of the price',
    `zip_code`        VARCHAR(10) DEFAULT NULL COMMENT 'Zip code for local prices (NULL for national)',
    `raw_response`    JSON DEFAULT NULL COMMENT 'Original JSON response from API',
    `fetched_at`      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'When the API was called',
    `created_at`      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Row creation timestamp',

    -- Unique constraint to prevent duplicate entries
    UNIQUE KEY `unique_provider_product_date` (`provider`, `product_type`, `price_date`, `zip_code`),

    -- Indexes for common queries
    INDEX `idx_price_date` (`price_date`),
    INDEX `idx_provider` (`provider`),
    INDEX `idx_product_type` (`product_type`),
    INDEX `idx_scope` (`scope`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
