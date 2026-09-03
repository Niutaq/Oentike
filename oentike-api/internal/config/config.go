package config

import "os"

const defaultDatabaseURL = "postgres://oentike:oentike@localhost:5432/oentike?sslmode=disable"

type Config struct {
	DatabaseURL  string
	HTTPAddr     string
	GRPCAddr     string
	OpenMeteoURL string
}

func Load() Config {
	return Config{
		DatabaseURL:  valueOrDefault("DATABASE_URL", defaultDatabaseURL),
		HTTPAddr:     valueOrDefault("HTTP_ADDR", ":8081"),
		GRPCAddr:     valueOrDefault("GRPC_ADDR", ":8082"),
		OpenMeteoURL: valueOrDefault("OPEN_METEO_URL", "https://api.open-meteo.com"),
	}
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
