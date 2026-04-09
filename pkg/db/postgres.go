package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	"example.com/klyntar-server/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func New(dbConfig config.DBConfig) (*sql.DB, error) {
	if dbConfig.DSN == "" {
		log.Fatalln("Cannot connect to DB, DSN not present")
	}

	dsn := dbConfig.DSN
	if !strings.Contains(dsn, "sslmode=") {
		dsn += "?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalln("cannot connect to DB", err)
	}

	db.SetMaxOpenConns(dbConfig.MaxOpenConns)
	db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	duration, err := time.ParseDuration("10m")
	if err != nil {
		slog.Error("Error setting db time limit")
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)

	context, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(context); err != nil {
		fmt.Printf("Cannot Ping Db %s %s \n", dsn, err)
	}

	return db, nil
}

func Migrate(dbDsn string, migrationPath string) error {
	m, err := migrate.New(migrationPath, dbDsn)
	if err != nil {
		return fmt.Errorf("migrate.New: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate.Up: %w", err)
	}

	return nil
}
