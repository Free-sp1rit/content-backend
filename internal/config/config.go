package config

import "time"

const (
	defaultServerPort    = "8080"
	defaultJWTIssuer     = "content-backend"
	defaultJWTTokenTTL   = 24 * time.Hour
	defaultReadHeaderTTL = 5 * time.Second
)

type Config struct {
}
