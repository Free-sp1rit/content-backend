package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultServerPort    = "8080"
	defaultReadHeaderTTL = 5 * time.Second

	defaultDBHost    = "127.0.0.1"
	defaultDBPort    = "5432"
	defaultDBSSLMode = "disable"

	defaultJWTIssuer   = "content-backend"
	defaultJWTTokenTTL = 24 * time.Hour

	defaultRedisAddr = "127.0.0.1:6379"
	defaultRedisDB   = 0
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Redis    RedisConfig
}

type ServerConfig struct {
	Port              string
	ReadHeaderTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type JWTConfig struct {
	Secret   string
	Issuer   string
	TokenTTL time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

func Load() (Config, error) {
	cfg := Config{
		Server: ServerConfig{
			Port:              defaultServerPort,
			ReadHeaderTimeout: defaultReadHeaderTTL,
		},
		Database: DatabaseConfig{
			Host:    defaultDBHost,
			Port:    defaultDBPort,
			SSLMode: defaultDBSSLMode,
		},
		JWT: JWTConfig{
			Issuer:   defaultJWTIssuer,
			TokenTTL: defaultJWTTokenTTL,
		},
		Redis: RedisConfig{
			Addr: defaultRedisAddr,
			DB:   defaultRedisDB,
		},
	}

	cfg.Server.Port = getEnv("PORT", defaultServerPort)
	readHeaderTimeout, err := getEnvDuration("READ_HEADER_TIMEOUT", defaultReadHeaderTTL)
	if err != nil {
		return Config{}, err
	}
	cfg.Server.ReadHeaderTimeout = readHeaderTimeout

	cfg.Database.Host = getEnv("DB_HOST", defaultDBHost)
	cfg.Database.Port = getEnv("DB_PORT", defaultDBPort)
	cfg.Database.User = getEnv("DB_USER", "")
	cfg.Database.Password = getEnv("DB_PASSWORD", "")
	cfg.Database.Name = getEnv("DB_NAME", "")
	cfg.Database.SSLMode = getEnv("DB_SSLMODE", defaultDBSSLMode)
	if cfg.Database.User == "" || cfg.Database.Password == "" || cfg.Database.Name == "" {
		return Config{}, errors.New("database config is incomplete")
	}

	cfg.JWT.Secret = getEnv("JWT_SECRET", "")
	cfg.JWT.Issuer = getEnv("JWT_ISSUER", defaultJWTIssuer)
	JWTTokenTTL, err := getEnvDuration("JWT_TOKEN_TTL", defaultJWTTokenTTL)
	if err != nil {
		return Config{}, err
	}
	cfg.JWT.TokenTTL = JWTTokenTTL

	if cfg.JWT.Secret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	cfg.Redis.Addr = getEnv("REDIS_ADDR", defaultRedisAddr)
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	redisDB, err := getEnvInt("REDIS_DB", defaultRedisDB)
	if err != nil {
		return Config{}, err
	}
	cfg.Redis.DB = redisDB

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	if value := os.Getenv(key); value != "" {
		d, err := time.ParseDuration(value)
		if err != nil {
			return 0, fmt.Errorf("parse %s: %w", key, err)
		}
		return d, nil
	}
	return defaultValue, nil
}

func getEnvInt(key string, defaultValue int) (int, error) {
	if value := os.Getenv(key); value != "" {
		i, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("parse %s: %w", key, err)
		}
		return i, nil
	}
	return defaultValue, nil
}
