-- Weather observations table for daily weather data from multiple providers.
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
