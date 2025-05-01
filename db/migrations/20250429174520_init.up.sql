CREATE TABLE IF NOT EXISTS books (
    id INTEGER PRIMARY KEY,
    isbn TEXT NOT NULL UNIQUE CHECK (LENGTH(TRIM(isbn)) > 0),
    title TEXT NOT NULL CHECK (LENGTH(TRIM(title)) > 0),
    author TEXT NOT NULL CHECK (LENGTH(TRIM(author)) > 0),
    description TEXT,
    checked_out_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    archived_at TIMESTAMP
);
--
CREATE INDEX idx_books_isbn ON books(isbn);
CREATE INDEX idx_books_title ON books(title);
CREATE INDEX idx_books_author ON books(author);
CREATE INDEX idx_books_checked_out_at ON books(checked_out_at);
--
--
CREATE TABLE IF NOT EXISTS book_events (
    id INTEGER PRIMARY KEY,
    book_id INTEGER NOT NULL CHECK (LENGTH(TRIM(book_id)) > 0) REFERENCES books(id) ON DELETE CASCADE,
    action TEXT NOT NULL CHECK(action IN ('checkin', 'checkout')),
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
--
CREATE INDEX idx_book_events_book_id ON book_events(book_id);
CREATE INDEX idx_book_events_timestamp ON book_events(timestamp);
