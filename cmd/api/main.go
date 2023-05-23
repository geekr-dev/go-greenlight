package main

import (
	"context"
	"database/sql"
	"flag"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"greenlight.geekr.dev/internal/data"
	"greenlight.geekr.dev/internal/jsonlog"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
}

type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://greenlight:paSSword@localhost/greenlight", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", false, "Rate limiter enabled")

	flag.Parse()

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	db, err := openDB(cfg)
	if err != nil {
		logger.Fatal(err, nil)
	}
	defer db.Close()

	logger.Info("database connection pool established", nil)

	migrationDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logger.Fatal(err, nil)
	}

	migrator, err := migrate.NewWithDatabaseInstance("file:///home/geekr/Development/go/greenlight/migrations", "postgres", migrationDriver)
	if err != nil {
		logger.Fatal(err, nil)
	}

	err = migrator.Up()
	if err != nil && err != migrate.ErrNoChange {
		logger.Fatal(err, nil)
	}

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

	err = app.serve()
	if err != nil {
		logger.Fatal(err, nil)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)

	// Ping the database to check it's still alive.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
