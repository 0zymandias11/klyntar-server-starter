package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	userspb "example.com/klyntar-server/gen/pb/users/v1"
	"example.com/klyntar-server/config"
	"example.com/klyntar-server/internal/users"
	"example.com/klyntar-server/pkg/cache"
	"example.com/klyntar-server/pkg/db"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	})))

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

	// gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %s", cfg.GRPCPort, err)
	}

	grpcServer := grpc.NewServer()

	// Register services
	usersHandler := users.NewHandler(sqlDB)
	userspb.RegisterUsersServiceServer(grpcServer, usersHandler)

	// Enable reflection for tools like grpcurl and evans
	reflection.Register(grpcServer)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("shutting down gRPC server")
		grpcServer.GracefulStop()
	}()

	slog.Info("gRPC server listening", "addr", lis.Addr().String())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server error: %s", err)
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
