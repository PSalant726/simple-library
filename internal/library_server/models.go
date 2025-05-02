package library_server

import (
	"database/sql"
	"encoding/json"
	"time"
)

type ArchiveBookRequest struct {
	ID         int64     `json:"id"`
	ArchivedAt time.Time `json:"archived_at"`
}

type Book struct {
	ID           int64          `json:"id"`
	Isbn         string         `json:"isbn"`
	Title        string         `json:"title"`
	Author       string         `json:"author"`
	Description  sql.NullString `json:"description"`
	CheckedOutAt sql.NullTime   `json:"checked_out_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	ArchivedAt   sql.NullTime   `json:"archived_at"`
}

type Response struct {
	Data    json.RawMessage `json:"data,omitempty"`
	Message string          `json:"message"`
}

type UpdateBookRequest struct {
	ID          int64  `json:"id"`
	Isbn        string `json:"isbn"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
}
