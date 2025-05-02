package library_server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	db "github.com/PSalant726/simple-library/db/sqlc"
	"modernc.org/sqlite"
)

const (
	bookActionCheckIn  = "checkin"
	bookActionCheckOut = "checkout"

	routeBooks = "/books"
)

var (
	endpointArchiveBook  = fmt.Sprintf("%s %s/archive", http.MethodPatch, routeBooks)
	endpointCheckInBook  = fmt.Sprintf("%s %s/{id}/check-in", http.MethodPatch, routeBooks)
	endpointCheckOutBook = fmt.Sprintf("%s %s/{id}/check-out", http.MethodPatch, routeBooks)
	endpointCreateBook   = fmt.Sprintf("%s %s", http.MethodPost, routeBooks)
	endpointDeleteBook   = fmt.Sprintf("%s %s/{id}", http.MethodDelete, routeBooks)
	endpointGetBook      = fmt.Sprintf("%s %s/{id}", http.MethodGet, routeBooks)
	endpointListBooks    = fmt.Sprintf("%s %s", http.MethodGet, routeBooks)
	endpointUpdateBook   = fmt.Sprintf("%s %s", http.MethodPut, routeBooks)
)

func (s *Server) addBooksRoutes(mux *http.ServeMux) {
	mux.HandleFunc(endpointArchiveBook, s.handleArchiveBook)
	mux.HandleFunc(endpointCheckInBook, s.handleCheckInBook)
	mux.HandleFunc(endpointCheckOutBook, s.handleCheckOutBook)
	mux.HandleFunc(endpointCreateBook, s.handleCreateBook)
	mux.HandleFunc(endpointDeleteBook, s.handleDeleteBook)
	mux.HandleFunc(endpointGetBook, s.handleGetBook)
	mux.HandleFunc(endpointListBooks, s.handleListBooks)
	mux.HandleFunc(endpointUpdateBook, s.handleUpdateBook)
}

func (s *Server) handleArchiveBook(w http.ResponseWriter, r *http.Request) {
	var archiveReq ArchiveBookRequest
	if code, err := strictDecode(w, r, &archiveReq); err != nil {
		s.logger.Warn(
			"Failed to decode request body",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, code, Response{
			Message: err.Error(),
		})
		return
	}

	if archiveReq.ArchivedAt.IsZero() {
		s.logger.Warn("Archive timestamp not provided", "request_id", r.Context().Value(requestID))
		respondWithJSON(w, http.StatusBadRequest, Response{
			Message: "Archive timestamp not provided.",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultDBTimeout)
	result, err := s.libraryDB.ArchiveBook(ctx, db.ArchiveBookParams{
		ID: archiveReq.ID,
		ArchivedAt: sql.NullTime{
			Time:  archiveReq.ArchivedAt,
			Valid: true,
		},
	})
	if errors.Is(err, sql.ErrNoRows) {
		respondWithJSON(w, http.StatusNotFound, Response{
			Message: fmt.Sprintf("Book with ID %d not found.", archiveReq.ID),
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

	jsonResult, err := json.Marshal(result)
	if err != nil {
		s.logger.Error(
			"Failed to marshal book as JSON",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}

	respondWithJSON(w, http.StatusOK, Response{
		Data:    jsonResult,
		Message: "Book archived.",
	})
}

func (s *Server) handleCheckInBook(w http.ResponseWriter, r *http.Request) {
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

	jsonResult, err := json.Marshal(result)
	if err != nil {
		s.logger.Error(
			"Failed to marshal book as JSON",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}

	respondWithJSON(w, http.StatusOK, Response{
		Data:    jsonResult,
		Message: "Book checked in.",
	})
}

func (s *Server) handleCheckOutBook(w http.ResponseWriter, r *http.Request) {
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

	jsonResult, err := json.Marshal(result)
	if err != nil {
		s.logger.Error(
			"Failed to marshal book as JSON",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}

	respondWithJSON(w, http.StatusOK, Response{
		Data:    jsonResult,
		Message: "Book checked out.",
	})
}

func (s *Server) handleCreateBook(w http.ResponseWriter, r *http.Request) {
	var book Book
	if code, err := strictDecode(w, r, &book); err != nil {
		s.logger.Warn(
			"Failed to decode request body",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, code, Response{
			Message: err.Error(),
		})
		return
	}

	if strings.TrimSpace(book.Isbn) == "" ||
		strings.TrimSpace(book.Title) == "" ||
		strings.TrimSpace(book.Author) == "" {
		s.logger.Warn(
			"Missing required field(s)",
			"request_id", r.Context().Value(requestID),
			slog.Group(
				"provided",
				"author", book.Author,
				"isbn", book.Isbn,
				"title", book.Title,
			),
		)
		respondWithJSON(w, http.StatusBadRequest, Response{
			Message: "ISBN, title, and author must not be empty.",
			Data: json.RawMessage(fmt.Sprintf(
				`{"author": %q, "isbn": %q, "title": %q}`,
				book.Author, book.Isbn, book.Title,
			)),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultDBTimeout)
	result, err := s.libraryDB.CreateBook(ctx, db.CreateBookParams{
		Isbn:         book.Isbn,
		Title:        book.Title,
		Author:       book.Author,
		Description:  book.Description,
		CheckedOutAt: book.CheckedOutAt,
		ArchivedAt:   book.ArchivedAt,
	})
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			s.logger.Warn(
				"Failed to insert book",
				"request_id", r.Context().Value(requestID),
				"error", sqliteErr,
			)
			handleSQLiteBookError(w, sqliteErr, Book{
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

	jsonResult, err := json.Marshal(result)
	if err != nil {
		s.logger.Error(
			"Failed to marshal response JSON",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}

	respondWithJSON(w, http.StatusCreated, Response{
		Data:    jsonResult,
		Message: "Book created.",
	})
}

func (s *Server) handleDeleteBook(w http.ResponseWriter, r *http.Request) {
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
	result, err := s.libraryDB.DeleteBook(ctx, bookID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithJSON(w, http.StatusNotFound, Response{
			Message: fmt.Sprintf("Book with ID %d not found.", bookID),
		})
		cancel()
		return
	}
	if err != nil {
		s.logger.Error(
			"Failed to delete book",
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

	respondWithJSON(w, http.StatusOK, Response{
		Message: fmt.Sprintf("Deleted book with ID %d.", result.ID),
	})
}

func (s *Server) handleGetBook(w http.ResponseWriter, r *http.Request) {
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
	result, err := s.libraryDB.Book(ctx, bookID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithJSON(w, http.StatusNotFound, Response{
			Message: fmt.Sprintf("Book with ID %d not found.", bookID),
		})
		cancel()
		return
	}
	if err != nil {
		s.logger.Error(
			"Failed to fetch book",
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

	jsonResult, err := json.Marshal(result)
	if err != nil {
		s.logger.Error(
			"Failed to marshal book as JSON",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}

	respondWithJSON(w, http.StatusOK, Response{
		Data:    jsonResult,
		Message: "Book found.",
	})
}

func (s *Server) handleListBooks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), defaultDBTimeout)
	results, err := s.libraryDB.Books(ctx)
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

	resp := Response{
		Data:    json.RawMessage(`[]`),
		Message: "Listed books.",
	}

	if len(results) > 0 {
		jsonResult, err := json.Marshal(results)
		if err != nil {
			s.logger.Error(
				"Failed to marshal books as JSON",
				"request_id", r.Context().Value(requestID),
				"error", err,
			)
			respondWithJSON(w, http.StatusInternalServerError, Response{
				Message: http.StatusText(http.StatusInternalServerError),
			})
			return
		}

		resp.Data = jsonResult
	}

	respondWithJSON(w, http.StatusOK, resp)
}

func (s *Server) handleUpdateBook(w http.ResponseWriter, r *http.Request) {
	var updateReq UpdateBookRequest
	if code, err := strictDecode(w, r, &updateReq); err != nil {
		s.logger.Warn(
			"Failed to decode request body",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, code, Response{
			Message: err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultDBTimeout)
	result, err := s.libraryDB.UpdateBook(ctx, db.UpdateBookParams{
		ID:          updateReq.ID,
		Isbn:        updateReq.Isbn,
		Title:       updateReq.Title,
		Author:      updateReq.Author,
		Description: updateReq.Description,
	})
	if errors.Is(err, sql.ErrNoRows) {
		respondWithJSON(w, http.StatusNotFound, Response{
			Message: fmt.Sprintf("Book with ID %d not found.", updateReq.ID),
		})
		cancel()
		return
	}
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			s.logger.Warn(
				"Failed to update book",
				"request_id", r.Context().Value(requestID),
				"error", sqliteErr,
			)
			handleSQLiteBookError(w, sqliteErr, Book{
				ID:     updateReq.ID,
				Isbn:   updateReq.Isbn,
				Title:  updateReq.Title,
				Author: updateReq.Author,
			})
			cancel()
			return
		}

		s.logger.Error(
			"Failed to update book",
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

	jsonResult, err := json.Marshal(result)
	if err != nil {
		s.logger.Error(
			"Failed to marshal response JSON",
			"request_id", r.Context().Value(requestID),
			"error", err,
		)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}

	respondWithJSON(w, http.StatusOK, Response{
		Data:    jsonResult,
		Message: "Book updated.",
	})
}
