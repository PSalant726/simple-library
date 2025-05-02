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
	"time"

	db "github.com/PSalant726/simple-library/db/sqlc"
	"modernc.org/sqlite"
)

const routeBookEvents = "/book_events"

var (
	endpointCreateBookEvent        = fmt.Sprintf("%s %s", http.MethodPost, routeBookEvents)
	endpointDeleteBookEvent        = fmt.Sprintf("%s %s/{id}", http.MethodDelete, routeBookEvents)
	endpointGetBookEvent           = fmt.Sprintf("%s %s/{id}", http.MethodGet, routeBookEvents)
	endpointListBookEvents         = fmt.Sprintf("%s %s", http.MethodGet, routeBookEvents)
	endpointListBookEventsByBookID = fmt.Sprintf("%s %s/book/{id}", http.MethodGet, routeBookEvents)
	endpointUpdateBookEvent        = fmt.Sprintf("%s %s", http.MethodPut, routeBookEvents)
)

func (s *Server) addBookEventsRoutes(mux *http.ServeMux) {
	mux.HandleFunc(endpointCreateBookEvent, s.handleCreateBookEvent)
	mux.HandleFunc(endpointDeleteBookEvent, s.handleDeleteBookEvent)
	mux.HandleFunc(endpointGetBookEvent, s.handleGetBookEvent)
	mux.HandleFunc(endpointListBookEvents, s.handleListBookEvents)
	mux.HandleFunc(endpointListBookEventsByBookID, s.handleListBookEventsByBookID)
	mux.HandleFunc(endpointUpdateBookEvent, s.handleUpdateBookEvent)
}

func (s *Server) handleCreateBookEvent(w http.ResponseWriter, r *http.Request) {
	var bookEvent BookEvent
	if code, err := strictDecode(w, r, &bookEvent); err != nil {
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

	if bookEvent.BookID == 0 ||
		(bookEvent.Action != bookActionCheckIn && bookEvent.Action != bookActionCheckOut) ||
		bookEvent.Timestamp.IsZero() {
		s.logger.Warn(
			"Missing or invalid field(s)",
			"request_id", r.Context().Value(requestID),
			slog.Group(
				"provided",
				"book_id", bookEvent.BookID,
				"action", bookEvent.Action,
				"timestamp", bookEvent.Timestamp.Format(time.RFC3339),
			),
		)
		respondWithJSON(w, http.StatusBadRequest, Response{
			Message: "Missing or invalid field(s).",
			Data: json.RawMessage(fmt.Sprintf(
				`{"book_id": %d, "action": %q, "timestamp": %q}`,
				bookEvent.BookID, bookEvent.Action, bookEvent.Timestamp.Format(time.RFC3339),
			)),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultDBTimeout)
	result, err := s.libraryDB.CreateBookEvent(ctx, db.CreateBookEventParams{
		BookID:    bookEvent.BookID,
		Action:    bookEvent.Action,
		Timestamp: bookEvent.Timestamp,
	})
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			s.logger.Error(
				"Failed to insert book event",
				"request_id", r.Context().Value(requestID),
				"error", sqliteErr,
			)
			handleSQLiteBookEventError(w, sqliteErr, BookEvent{
				BookID: bookEvent.BookID,
				Action: bookEvent.Action,
			})
			cancel()
			return
		}

		s.logger.Error(
			"Failed to insert book event",
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
		Message: "Book event created.",
	})
}

func (s *Server) handleDeleteBookEvent(w http.ResponseWriter, r *http.Request) {
	bookEventID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.logger.Warn(
			"Invalid id value",
			"id", r.PathValue("id"),
			"request_id", r.Context().Value(requestID),
		)
		respondWithJSON(w, http.StatusBadRequest, Response{
			Message: "Invalid book event ID. Must be an integer.",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultDBTimeout)
	result, err := s.libraryDB.DeleteBookEvent(ctx, bookEventID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithJSON(w, http.StatusNotFound, Response{
			Message: fmt.Sprintf("Book event with ID %d not found.", bookEventID),
		})
		cancel()
		return
	}
	if err != nil {
		s.logger.Error(
			"Failed to delete book event",
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
		Message: fmt.Sprintf("Deleted book event with ID %d.", result.ID),
	})
}

func (s *Server) handleGetBookEvent(w http.ResponseWriter, r *http.Request) {
	bookEventID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.logger.Warn(
			"Invalid id value",
			"id", r.PathValue("id"),
			"request_id", r.Context().Value(requestID),
		)
		respondWithJSON(w, http.StatusBadRequest, Response{
			Message: "Invalid book event ID. Must be an integer.",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultDBTimeout)
	result, err := s.libraryDB.BookEvent(ctx, bookEventID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithJSON(w, http.StatusNotFound, Response{
			Message: fmt.Sprintf("Book event with ID %d not found.", bookEventID),
		})
		cancel()
		return
	}
	if err != nil {
		s.logger.Error(
			"Failed to fetch book event",
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
			"Failed to marshal book event as JSON",
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
		Message: "Book event found.",
	})
}

func (s *Server) handleListBookEvents(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), defaultDBTimeout)
	results, err := s.libraryDB.BookEvents(ctx)
	if err != nil {
		s.logger.Error(
			"Failed to fetch book events",
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
		Message: "Listed book events.",
	}

	if len(results) > 0 {
		jsonResult, err := json.Marshal(results)
		if err != nil {
			s.logger.Error(
				"Failed to marshal book events as JSON",
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

func (s *Server) handleListBookEventsByBookID(w http.ResponseWriter, r *http.Request) {
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
	results, err := s.libraryDB.BookEventsByBookID(ctx, bookID)
	if err != nil {
		s.logger.Error(
			"Failed to fetch book events",
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
		Message: fmt.Sprintf("Listed book events with book ID %d.", bookID),
	}

	if len(results) > 0 {
		jsonResult, err := json.Marshal(results)
		if err != nil {
			s.logger.Error(
				"Failed to marshal book events as JSON",
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

func (s *Server) handleUpdateBookEvent(w http.ResponseWriter, r *http.Request) {
	var updateReq UpdateBookEventRequest
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

	if updateReq.ID == 0 ||
		updateReq.BookID == 0 ||
		(updateReq.Action != bookActionCheckIn && updateReq.Action != bookActionCheckOut) {
		s.logger.Warn(
			"Missing or invalid field(s)",
			"request_id", r.Context().Value(requestID),
			slog.Group(
				"provided",
				"id", updateReq.ID,
				"book_id", updateReq.BookID,
				"action", updateReq.Action,
				"timestamp", updateReq.Timestamp.Format(time.RFC3339),
			),
		)
		respondWithJSON(w, http.StatusBadRequest, Response{
			Message: "Missing or invalid field(s).",
			Data: json.RawMessage(fmt.Sprintf(
				`{"id": %d, "book_id": %d, "action": %q, "timestamp": %q}`,
				updateReq.ID, updateReq.BookID, updateReq.Action, updateReq.Timestamp.Format(time.RFC3339),
			)),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultDBTimeout)
	result, err := s.libraryDB.UpdateBookEvent(ctx, db.UpdateBookEventParams{
		ID:                    updateReq.ID,
		BookID:                updateReq.BookID,
		Action:                updateReq.Action,
		ShouldUpdateTimestamp: !updateReq.Timestamp.IsZero(),
		Timestamp:             updateReq.Timestamp,
	})
	if errors.Is(err, sql.ErrNoRows) {
		respondWithJSON(w, http.StatusNotFound, Response{
			Message: fmt.Sprintf("Book event with ID %d not found.", updateReq.ID),
		})
		cancel()
		return
	}
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			s.logger.Warn(
				"Failed to update book event",
				"request_id", r.Context().Value(requestID),
				"error", sqliteErr,
			)
			handleSQLiteBookEventError(w, sqliteErr, BookEvent{
				ID:     updateReq.ID,
				BookID: updateReq.BookID,
				Action: updateReq.Action,
			})
			cancel()
			return
		}

		s.logger.Error(
			"Failed to update book event",
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
		Message: "Book event updated.",
	})
}
