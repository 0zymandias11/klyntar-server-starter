package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"example.com/klyntar-server/api/routes"
	"example.com/klyntar-server/app"
	"example.com/klyntar-server/config"
	"example.com/klyntar-server/pkg/cache"
	"example.com/klyntar-server/pkg/db"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	})))

	// Find the project root: .env and migrations/ live here.
	// Works whether you run from the project root or the server/ subdir.
	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("failed to find project root: %s", err)
	}
	if err := os.Chdir(projectRoot); err != nil {
		log.Fatalf("failed to chdir to project root: %s", err)
	}

	if err := godotenv.Load(".env"); err != nil {
		slog.Warn("could not load .env file", "error", err)
	}

	cfg := config.Load()

	slog.Info("klyntar-server starting", "grpc_port", cfg.GRPCPort)

	// Connect to Postgres
	sqlDB, err := db.New(cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %s", err)
	}
	defer sqlDB.Close()
	slog.Info("connected to postgres")

	// Connect to Redis
	redisClient, err := cache.New(cfg.Redis)
	if err != nil {
		log.Fatalf("failed to connect to redis: %s", err)
	}
	defer redisClient.Close()
	slog.Info("connected to redis", "addr", cfg.Redis.Addr)

	if err := db.Migrate(cfg.DB.DSN, "file://migrations"); err != nil {
		slog.Error("unable to run db migrations", "error", err)
	}
	application := &app.Application{
		Config:      cfg,
		DbConnector: sqlDB,
	}

	application.RegisterRoutes(
		routes.RegisterUserRoutes,
	)

	application.RegisterSoloRoutes(
		routes.RegisterHealthRoutes,
		
	)

	mux := application.Mount()
	if err := application.Run(mux); err != nil {
		log.Fatalf("Error Starting new server : %s", err)
	}
}

// findProjectRoot walks up from the working directory looking for go.mod.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod in any parent directory")
		}
		dir = parent
	}
}
