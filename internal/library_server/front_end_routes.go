package library_server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	db "github.com/PSalant726/simple-library/db/sqlc"
	"modernc.org/sqlite"
)

const (
	routeUIBooks       = "/ui/books"
	routeUICheckOutLog = "/check-out-log"
)

var (
	endpointIndex          = http.MethodGet + " /"
	endpointUICreateBook   = fmt.Sprintf("%s %s", http.MethodPost, routeUIBooks)
	endpointUICheckInBook  = fmt.Sprintf("%s %s/{id}/check-in", http.MethodPatch, routeUIBooks)
	endpointUICheckOutBook = fmt.Sprintf("%s %s/{id}/check-out", http.MethodPatch, routeUIBooks)
	endpointUIDeleteBook   = fmt.Sprintf("%s %s/{id}", http.MethodDelete, routeUIBooks)
	endpointUICheckOutLog  = fmt.Sprintf("%s %s", http.MethodGet, routeUICheckOutLog)
)

func (s *Server) addFrontEndRoutes(mux *http.ServeMux) {
	mux.HandleFunc(endpointIndex, s.handleIndex)
	mux.HandleFunc(endpointUICreateBook, s.handleUICreateBook)
	mux.HandleFunc(endpointUICheckInBook, s.handleUICheckInBook)
	mux.HandleFunc(endpointUICheckOutBook, s.handleUICheckOutBook)
	mux.HandleFunc(endpointUIDeleteBook, s.handleUIDeleteBook)
	mux.HandleFunc(endpointUICheckOutLog, s.handleUICheckOutLog)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	books, err := s.libraryDB.Books(ctx)
	if err != nil {
		s.logger.Error(
			"Failed to fetch books",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		cancel()
		return
	}
	cancel()

	tmpl, err := template.ParseFiles("templates/index.html", "templates/base.html")
	if err != nil {
		s.logger.Error(
			"Failed to parse templates",
			"request_id", r.Context().Value(requestID),
			"templates", tmpl.Name,
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}

	if err := tmpl.ExecuteTemplate(
		w,
		"base.html",
		map[string]any{"Books": books},
	); err != nil {
		s.logger.Error(
			"Failed to execute template",
			"request_id", r.Context().Value(requestID),
			"template", tmpl.Name,
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}
}

func (s *Server) handleUICreateBook(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.logger.Warn(
			"Failed to parse form",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusBadRequest, Response{
			Message: "Failed to parse book details.",
		})
		return
	}

	book := Book{
		Isbn:   strings.TrimSpace(r.FormValue("isbn")),
		Title:  strings.TrimSpace(r.FormValue("title")),
		Author: strings.TrimSpace(r.FormValue("author")),
		Description: sql.NullString{
			String: strings.TrimSpace(r.FormValue("description")),
		},
	}
	book.Description.Valid = book.Description.String != ""

	ctx, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	result, err := s.libraryDB.CreateBook(ctx, db.CreateBookParams{
		Isbn:        book.Isbn,
		Title:       book.Title,
		Author:      book.Author,
		Description: book.Description,
	})
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			s.logger.Warn(
				"Failed to insert book",
				"request_id", r.Context().Value(requestID),
				"error", sqliteErr,
			)
			handleSQLiteBookUIError(w, sqliteErr, Book{
				Isbn:   book.Isbn,
				Title:  book.Title,
				Author: book.Author,
			})
			cancel()
			return
		}

		s.logger.Error(
			"Failed to insert book",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		cancel()
		return
	}
	cancel()

	tmpl, err := template.ParseFiles("templates/book.html")
	if err != nil {
		s.logger.Error(
			"Failed to parse template",
			"request_id", r.Context().Value(requestID),
			"template", tmpl.Name,
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}

	if err := tmpl.Execute(w, result); err != nil {
		s.logger.Error(
			"Failed to execute template",
			"request_id", r.Context().Value(requestID),
			"template", tmpl.Name,
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}
}

func (s *Server) handleUICheckInBook(w http.ResponseWriter, r *http.Request) {
	bookID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.logger.Warn(
			"Invalid id value",
			"id", r.PathValue("id"),
			"request_id", r.Context().Value(requestID),
		)
		respondWithJSON(w, http.StatusBadRequest, Response{
			Message: "Invalid book ID. Must be an integer.",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultDBTimeout)
	tx, err := s.libraryDB.DB.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error(
			"Failed to begin database transaction",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		cancel()
		return
	}
	defer tx.Rollback()

	checkinTx := s.libraryDB.WithTx(tx)

	result, err := checkinTx.CheckInBook(ctx, bookID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithJSON(w, http.StatusNotFound, Response{
			Message: fmt.Sprintf("Book with ID %d not found.", bookID),
		})
		cancel()
		return
	}
	if err != nil {
		s.logger.Error(
			"Failed to check-in book",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		cancel()
		return
	}

	if _, err := checkinTx.CreateBookEvent(ctx, db.CreateBookEventParams{
		BookID:    bookID,
		Action:    bookActionCheckIn,
		Timestamp: time.Now().UTC(),
	}); err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			s.logger.Warn(
				"Failed to check-in book",
				"request_id", r.Context().Value(requestID),
				"error", sqliteErr,
			)
			handleSQLiteBookError(w, sqliteErr, Book{ID: bookID})
			cancel()
			return
		}

		s.logger.Error(
			"Failed to check-in book",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		cancel()
		return
	}
	if err := tx.Commit(); err != nil {
		s.logger.Error(
			"Failed to commit database transaction",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		cancel()
		return
	}
	cancel()

	tmpl, err := template.ParseFiles("templates/book.html")
	if err != nil {
		s.logger.Error(
			"Failed to parse template",
			"request_id", r.Context().Value(requestID),
			"template", tmpl.Name,
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}

	if err := tmpl.Execute(w, result); err != nil {
		s.logger.Error(
			"Failed to execute template",
			"request_id", r.Context().Value(requestID),
			"template", tmpl.Name,
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}
}

func (s *Server) handleUICheckOutBook(w http.ResponseWriter, r *http.Request) {
	bookID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.logger.Warn(
			"Invalid id value",
			"id", r.PathValue("id"),
			"request_id", r.Context().Value(requestID),
		)
		respondWithJSON(w, http.StatusBadRequest, Response{
			Message: "Invalid book ID. Must be an integer.",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	tx, err := s.libraryDB.DB.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error(
			"Failed to begin database transaction",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		cancel()
		return
	}
	defer tx.Rollback()

	checkoutTx := s.libraryDB.WithTx(tx)

	now := time.Now().UTC()
	result, err := checkoutTx.CheckOutBook(ctx, db.CheckOutBookParams{
		ID: bookID,
		CheckedOutAt: sql.NullTime{
			Time:  now,
			Valid: true,
		},
	})
	if errors.Is(err, sql.ErrNoRows) {
		respondWithJSON(w, http.StatusNotFound, Response{
			Message: fmt.Sprintf("Book with ID %d not found.", bookID),
		})
		cancel()
		return
	}
	if err != nil {
		s.logger.Error(
			"Failed to check-out book",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		cancel()
		return
	}

	if _, err := checkoutTx.CreateBookEvent(ctx, db.CreateBookEventParams{
		BookID:    bookID,
		Action:    bookActionCheckOut,
		Timestamp: now,
	}); err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			s.logger.Warn(
				"Failed to check-out book",
				"request_id", r.Context().Value(requestID),
				"error", sqliteErr,
			)
			handleSQLiteBookError(w, sqliteErr, Book{ID: bookID})
			cancel()
			return
		}

		s.logger.Error(
			"Failed to check-out book",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		cancel()
		return
	}
	if err := tx.Commit(); err != nil {
		s.logger.Error(
			"Failed to commit database transaction",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		cancel()
		return
	}
	cancel()

	tmpl, err := template.ParseFiles("templates/book.html")
	if err != nil {
		s.logger.Error(
			"Failed to parse template",
			"request_id", r.Context().Value(requestID),
			"template", tmpl.Name,
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}

	if err := tmpl.Execute(w, result); err != nil {
		s.logger.Error(
			"Failed to execute template",
			"request_id", r.Context().Value(requestID),
			"template", tmpl.Name,
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}
}

func (s *Server) handleUIDeleteBook(w http.ResponseWriter, r *http.Request) {
	bookID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.logger.Warn(
			"Invalid id value",
			"id", r.PathValue("id"),
			"request_id", r.Context().Value(requestID),
		)
		respondWithJSON(w, http.StatusBadRequest, Response{
			Message: "Invalid book ID. Must be an integer.",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	_, err = s.libraryDB.ArchiveBook(ctx, db.ArchiveBookParams{
		ID: bookID,
		ArchivedAt: sql.NullTime{
			Time:  time.Now().UTC(),
			Valid: true,
		},
	})
	if errors.Is(err, sql.ErrNoRows) {
		respondWithJSON(w, http.StatusNotFound, Response{
			Message: fmt.Sprintf("Book with ID %d not found.", bookID),
		})
		cancel()
		return
	}
	if err != nil {
		s.logger.Error(
			"Failed to archive book",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		cancel()
		return
	}
	cancel()

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleUICheckOutLog(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), defaultDBTimeout)
	results, err := s.libraryDB.CheckOutLog(ctx)
	if err != nil {
		s.logger.Error(
			"Failed to fetch check-out log",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		cancel()
		return
	}
	cancel()

	tmpl, err := template.
		New("check_out_log.html").
		Funcs(template.FuncMap{
			"formatAction":    formatAction,
			"formatTimestamp": formatTimestamp,
		}).
		ParseFiles(
			"templates/check_out_log.html",
			"templates/base.html",
		)
	if err != nil {
		s.logger.Error(
			"Failed to parse template",
			"request_id", r.Context().Value(requestID),
			"template", tmpl.Name,
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}

	if err := tmpl.Execute(w, map[string]any{"CheckOutLog": results}); err != nil {
		s.logger.Error(
			"Failed to execute template",
			"request_id", r.Context().Value(requestID),
			"template", tmpl.Name,
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}
}
