package library_server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const (
	headerKeyContentType       = "Content-Type"
	headerValueApplicationJSON = "application/json"

	maxRequestBodySize = 1 << 20 // 1 MB
)

func handleSQLiteBookError(w http.ResponseWriter, err *sqlite.Error, book Book) {
	var resp Response

	switch err.Code() {
	case sqlite3.SQLITE_CONSTRAINT_CHECK:
		resp.Data = json.RawMessage(fmt.Sprintf(
			`{"author": %q, "isbn": %q, "title": %q}`,
			book.Author, book.Isbn, book.Title,
		))
		resp.Message = "ISBN, title, and author must not be empty."
	case sqlite3.SQLITE_CONSTRAINT_UNIQUE:
		resp.Message = fmt.Sprintf("A book with ISBN %q already exists.", book.Isbn)
	}

	respondWithJSON(w, http.StatusBadRequest, resp)
}

func respondWithJSON(w http.ResponseWriter, statusCode int, body Response) {
	w.Header().Set(headerKeyContentType, headerValueApplicationJSON)
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("Failed to encode response JSON", "error", err)
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Message: http.StatusText(http.StatusInternalServerError),
		})
		return
	}
}

// strictDecode uses best practices to safely unmarshal r's Body into v.
// When the returned error is non-nil, it also returns an HTTP status code
// suitable for use in a response.
func strictDecode(w http.ResponseWriter, r *http.Request, v any) (int, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		var syntaxErr *json.SyntaxError
		var unmarshalErr *json.UnmarshalTypeError
		var maxBytesErr *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxErr):
			return http.StatusBadRequest, fmt.Errorf("failed to parse JSON: %w", err)
		case errors.As(err, &unmarshalErr):
			return http.StatusBadRequest, fmt.Errorf("failed to unmarshal JSON into %T: %w", v, err)
		case errors.As(err, &maxBytesErr):
			return http.StatusRequestEntityTooLarge, err
		case errors.Is(err, io.EOF):
			return http.StatusBadRequest, errors.New("empty request body")
		default:
			return http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err)
		}
	}

	return 0, nil
}
