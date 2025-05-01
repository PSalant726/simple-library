package library_db

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"

	db "github.com/PSalant726/simple-library/db/sqlc"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	cfgBusyTimeout = "5000"   // milliseconds
	cfgCacheSize   = "64000"  // 64MB
	cfgJournalMode = "WAL"    // Write-Ahead Logging mode, for better concurrency
	cfgSynchronous = "NORMAL" // 2-3x faster than FULL, but still safe

	databaseName = "library"
)

type LibraryDB struct {
	*db.Queries

	DB                *sql.DB
	pathDBFile        string
	pathMigrationsDir string

	busyTimeout string
	cacheSize   string
	journalMode string
	name        string
	synchronous string
}

func New(dbFilePath, migrationsDirPath string) (*LibraryDB, error) {
	dbFile, err := os.OpenFile(dbFilePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open or create database file: %w", err)
	}
	defer dbFile.Close()

	if dir, err := os.Stat(migrationsDirPath); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("migrations directory path %q: %w", migrationsDirPath, err)
	} else if !dir.IsDir() {
		return nil, fmt.Errorf("migrations directory path %q is not a directory", migrationsDirPath)
	}

	return &LibraryDB{
		busyTimeout:       cfgBusyTimeout,
		cacheSize:         cfgCacheSize,
		journalMode:       cfgJournalMode,
		name:              databaseName,
		pathDBFile:        dbFilePath,
		pathMigrationsDir: migrationsDirPath,
		synchronous:       cfgSynchronous,
	}, nil
}

func (l *LibraryDB) Start() error {
	conn, err := sql.Open("sqlite", l.dataSourceName())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	l.Queries = db.New(conn)
	l.DB = conn

	if err := l.migrateUp(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

func (l LibraryDB) Stop() error {
	if err := l.DB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	return nil
}

func (l LibraryDB) dataSourceName() string {
	config := url.Values{}
	if l.busyTimeout != "" {
		config.Add("_busy_timeout", l.busyTimeout)
	}
	if l.cacheSize != "" {
		config.Add("_cache_size", l.cacheSize)
	}
	if l.journalMode != "" {
		config.Add("_journal_mode", l.journalMode)
	}
	if l.synchronous != "" {
		config.Add("_synchronous", l.synchronous)
	}

	config.Add("_pragma", "foreign_keys(1)")

	return fmt.Sprintf("file:%s?%s", l.pathDBFile, config.Encode())
}

func (l LibraryDB) migrateUp() error {
	driver, err := sqlite.WithInstance(
		l.DB,
		&sqlite.Config{DatabaseName: l.name},
	)
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", l.pathMigrationsDir),
		l.name,
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to execute up migrations: %w", err)
	}

	return nil
}
