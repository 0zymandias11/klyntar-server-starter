package config

import (
	"os"
	"strconv"
)

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type DBConfig struct {
	DSN          string
	MaxIdleConns int
	MaxOpenConns int
}

type Config struct {
	DB             DBConfig
	Redis          RedisConfig
	GRPCPort       int
	MigrationsPath string
	Addr           string
}

func Load() Config {
	return Config{
		DB: DBConfig{
			DSN:          getEnv("DATABASE_URL", "postgres://klyntar:klyntar@localhost:5432/klyntar?sslmode=disable"),
			MaxIdleConns: getInt("DB_MAX_IDLE_CONNS", 30),
			MaxOpenConns: getInt("DB_MAX_OPEN_CONNS", 30),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_URL", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getInt("REDIS_DB", 0),
		},
		GRPCPort:       getInt("GRPC_PORT", 50051),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "file://migrations"),
		Addr:           getEnv("Addr", ":3000"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
