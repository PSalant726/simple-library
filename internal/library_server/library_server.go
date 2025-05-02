package library_server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/PSalant726/simple-library/internal/library_db"
)

const (
	defaultDBTimeout     = 30 * time.Second
	defaultServerTimeout = 5 * time.Second
)

type Server struct {
	http.Server

	libraryDB *library_db.LibraryDB
	logger    slog.Logger
}

func New(host string, port int, dbFilePath, dbMigrationsDir string, verbose bool) (*Server, error) {
	libraryDB, err := library_db.New(dbFilePath, dbMigrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create library database: %w", err)
	}

	server := &Server{
		Server: http.Server{
			Addr: fmt.Sprintf("%s:%d", host, port),
			ErrorLog: slog.NewLogLogger(
				slog.Default().Handler().WithAttrs(
					[]slog.Attr{{
						Key:   "service",
						Value: slog.StringValue("library_server"),
					}},
				),
				slog.LevelError,
			),
			ReadHeaderTimeout: defaultServerTimeout,
			ReadTimeout:       defaultServerTimeout,
			WriteTimeout:      defaultServerTimeout,
		},
		libraryDB: libraryDB,
		logger:    *slog.Default().With("service", "library_server"),
	}

	server.addRoutes()
	return server, nil
}

func (s *Server) Start() error {
	s.logger.Debug("Creating library database connection...")
	if err := s.libraryDB.Start(); err != nil {
		return fmt.Errorf("failed to connect to library database: %w", err)
	}
	s.logger.Debug("Library database connection established")

	s.logger.Info("Listening for requests...", "server_address", s.Addr)
	if err := s.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to listen: %w", err)
	}

	return nil
}

func (s *Server) Stop(idleConnsChan chan struct{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultServerTimeout)
	defer cancel()

	s.logger.Debug("Gracefully stopping the library server...")
	if err := s.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to gracefully stop the library server: %w", err)
	}
	s.logger.Debug("Library server stopped")

	s.logger.Debug("Closing library database connection...")
	if err := s.libraryDB.Stop(); err != nil {
		return fmt.Errorf("failed to close library database connection: %w", err)
	}
	s.logger.Debug("Library database connection closed")

	close(idleConnsChan)
	return nil
}

func (s *Server) addRoutes() {
	mux := http.NewServeMux()
	s.addBooksRoutes(mux)
	s.addBookEventsRoutes(mux)
	s.addFrontEndRoutes(mux)
	s.Server.Handler = assignRequestID(logRequests(mux))
}
