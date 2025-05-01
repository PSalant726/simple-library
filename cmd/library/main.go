package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/PSalant726/simple-library/internal/library_server"
)

func main() {
	host := flag.String("host", "localhost", "The hostname of this server")
	port := flag.Int("port", 8080, "The port on which to listen")
	dbFilePath := flag.String("db-file", "db/library.sqlite", "The path to the SQLite database file")
	dbMigrationsDir := flag.String("db-migrations", "db/migrations", "The path to the SQLite database migrations directory")
	verboseLogging := flag.Bool("v", false, "Enable verbose logging")
	flag.Parse()

	if *verboseLogging {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	srv, err := library_server.New(*host, *port, *dbFilePath, *dbMigrationsDir, *verboseLogging)
	if err != nil {
		slog.Error("Failed to create library server", "error", err)
		os.Exit(1)
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		exitSigChan := make(chan os.Signal, 1)
		signal.Notify(exitSigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-exitSigChan
		slog.Debug("Shutdown signal received", "signal", sig)

		if err := srv.Stop(idleConnsClosed); err != nil {
			slog.Error("Failed to stop library server", "error", err)
		}
	}()

	slog.Info("Starting the library server", "host", *host, "port", *port)
	if err := srv.Start(); err != nil {
		slog.Error("Failed to start library server", "error", err)
		os.Exit(1)
	}

	<-idleConnsClosed
	slog.Info("Library server stopped. Goodbye!")
}
