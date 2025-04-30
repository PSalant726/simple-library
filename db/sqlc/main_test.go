package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

var testQueries *Queries

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "testmain")
	if err != nil {
		log.Printf("Failed to create temporary directory: %v", err)
		os.Exit(1)
	}

	conn, err := sql.Open("sqlite", filepath.Join(tmpDir, "library.test.db"))
	if err != nil {
		log.Printf("Failed to create test database connection: %v", err)
		os.Exit(1)
	}

	if _, err := conn.Exec("PRAGMA foreign_keys = ON;"); err != nil { // Enable foreign key enforcement
		log.Printf("Failed to enable foreign key constraints: %v", err)
		os.Exit(1)
	}

	driver, err := sqlite.WithInstance(conn, &sqlite.Config{})
	if err != nil {
		log.Printf("Failed to create test database driver: %v", err)
		os.Exit(1)
	}

	mi, err := migrate.NewWithDatabaseInstance(
		"file://../migrations",
		"test_library_db",
		driver,
	)
	if err != nil {
		log.Printf("Failed to create migration instance: %v", err)
		os.Exit(1)
	}

	if err := mi.Up(); err != nil { // Prepares DB schema & implicitly tests UP migrations
		log.Printf("Failed to run up migrations: %v", err)
		os.Exit(1)
	}

	testQueries = New(conn)
	exitCode := m.Run()

	if err := mi.Down(); err != nil { // Implicitly tests DOWN migrations
		log.Printf("Failed to run down migrations: %v", err)
		os.Exit(1)
	}

	conn.Close()
	os.RemoveAll(tmpDir)
	os.Exit(exitCode)
}
