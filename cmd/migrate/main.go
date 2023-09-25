package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	repository "github.com/serg666/service-repository"
	"github.com/serg666/ecom-service/config"
)

func main() {
	cfgPath, err := config.ParseFlags()
	if err != nil {
		log.Fatalf("can not parse flags due to: %v", err)
	}

	cfg, err := config.NewConfig(cfgPath)
	if err != nil {
		log.Fatalf("can not get new config due to: %v", err)
	}

	pgPool, err := repository.MakePgPoolFromDSN(cfg.Databases.Default.Dsn)
	if err != nil {
		log.Fatalf("Can not make pg pool: %v", err)
	}

	pgConf := pgPool.Config().ConnConfig.Config
	sslMode := "disable"
	if pgConf.TLSConfig != nil {
		sslMode = "require"
	}

	dbSourceName := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		pgConf.User, pgConf.Password, pgConf.Host, pgConf.Port, pgConf.Database, sslMode,
	)

	db, err := sql.Open("postgres", dbSourceName)
	if err != nil {
		log.Fatalf("Open postgres connection failed: %v", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("Open postgres driver failed: %v", err)
	}

	// Убрать после того как прод добавит конфиг "migration_source"
	dbMigrationSource := cfg.Databases.MigrationSource
	if dbMigrationSource == "" {
		dbMigrationSource = "file://migrations"
	}

	m, err := migrate.NewWithDatabaseInstance(dbMigrationSource, "postgres", driver)
	if err != nil {
		log.Fatalf("create migration instance failed: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration failed: %v", err)
	} else {
		log.Printf("Migration success")
	}
}
