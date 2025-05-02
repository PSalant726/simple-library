# Simple Library

## Overview

The Simple Library server is a web application designed to manage a library's books and their check-in/check-out events. It provides both a web-based user interface and a RESTful API for programmatic access.

## Original Prompt

> Using the stack of your choice, create a simple Library Management System that meets the following requirements:
> - Build an API to perform CRUD operations for books (fields: title, author, ISBN, description).
> - Provide a web interface for managing books.
> - Support checking books in and out, and tracking state changes over time.
> - Include a report that shows the current state of all books.


## Features

- Manage books (create, update, delete, archive)
- Track book check-in and check-out events
- View a log of all check-in and check-out events
- Web-based user interface for library management
- RESTful API for integration with other systems

## Prerequisites

- Go 1.24 or later
- SQLite 3
- Migration files for the database schema

## Usage

The server can be started in a dedicated terminal instance. By default, it listens at `http://localhost:8080`. From the repository's root directory, run:

```sh
go run ./cmd/library/...
```

Or, if you prefer to compile a binary instead:

```sh
make build
# or
go build -o ./bin/library ./cmd/library/...

./bin/library
```

To see all available options, use the `-h` flag:

```
$ ./bin/library -h
Usage of ./bin/library:
  -db-file string
        The path to the SQLite database file (default "db/library.sqlite")
  -db-migrations string
        The path to the SQLite database migrations directory (default "db/migrations")
  -host string
        The hostname of this server (default "localhost")
  -port int
        The port on which to listen (default 8080)
  -v    Enable verbose logging
```

> [!NOTE]
> If the SQLite database file does not already exist at the the location specified via the `-db-file` parameter, then one will be created automatically when the server starts.

## Testing

The included unit tests are somewhat robust, but test coverage is currently incomplete. They are only included as examples of my ability to write clear tests.

Run the unit tests with Go's built-in test runner:

```sh
make test
# or
go test -count=1 -cover -race ./...
```

## API Documentation

> [!IMPORTANT]
> 1. All timestamps must be in RFC3339 format (e.g., `2006-01-02T15:04:05Z` for UTC).
> 2. All returned timestamps use the UTC timezone.
> 3. Request bodies are limited to 1MB in size.
> 4. The API enforces strict JSON decoding - unknown fields will cause an error.
> 5. All responses use a consistent format with optional `data` and required `message` fields.
> 6. Database operations have a 30-second timeout.

### Base URL

All endpoints are relative to the base URL of the server (`http://localhost:8080` by default).

### Data Types

#### Book Object

```json
{
  "id": 123,
  "isbn": "978-3-16-148410-0",
  "title": "Book Title",
  "author": "Author Name",
  "description": "Optional description",
  "checked_out_at": "2006-01-02T15:04:05Z",  // nullable
  "archived_at": "2006-01-02T15:04:05Z",     // nullable
  "created_at": "2006-01-02T15:04:05Z",
  "updated_at": "2006-01-02T15:04:05Z"
}
```

#### Book Event Object

```json
{
  "id": 456,
  "book_id": 123,
  "action": "checkin",  // or "checkout"
  "timestamp": "2006-01-02T15:04:05Z",
  "created_at": "2006-01-02T15:04:05Z",
  "updated_at": "2006-01-02T15:04:05Z"
}
```

### Endpoints

#### Books

##### Create Book

**Endpoint:** `POST /books`

**Request Body:**

```json
{
  "isbn": "978-3-16-148410-0",
  "title": "Book Title",
  "author": "Author Name",
  "description": "Optional description"
}
```

**Response:** `201 Created`

```json
{
  "data": {Book},
  "message": "Book created"
}
```

**Error Responses:**

- `400 Bad Request`: Invalid JSON or missing required fields
- `413 Request Entity Too Large`: Request body exceeds 1MB
- `500 Internal Server Error`

##### Get Book

**Endpoint:** `GET /books/{id}`

**Path Parameters:**

- `id`: Integer ID of the book

**Response:** `200 OK`

```json
{
  "data": {Book},
  "message": "Book found"
}
```

**Error Responses:**

- `400 Bad Request`: Invalid ID format
- `404 Not Found`: Book ID not found
- `500 Internal Server Error`

##### List Books

**Endpoint:** `GET /books`

**Response:** `200 OK`

```json
{
  "data": [
    {Book},
    {Book},
    ...
  ],
  "message": "Listed books"
}
```

**Error Responses:**

- `500 Internal Server Error`

##### Update Book

**Endpoint:** `PUT /books`

**Request Body:**

```json
{
  "id": 123,
  "isbn": "978-3-16-148410-0",
  "title": "Updated Title",
  "author": "Updated Author",
  "description": "Updated description"
}
```

**Response:** `200 OK`

```json
{
  "data": {Book},
  "message": "Book updated"
}
```

**Error Responses:**

- `400 Bad Request`: Invalid JSON or missing required fields
- `404 Not Found`: Book ID not found
- `500 Internal Server Error`

##### Delete Book

**Endpoint:** `DELETE /books/{id}`

**Path Parameters:**

- `id`: Integer ID of the book

**Response:** `200 OK`

```json
{
  "message": "Deleted book with ID {id}"
}
```

**Error Responses:**

- `400 Bad Request`: Invalid ID format
- `404 Not Found`: Book ID not found
- `500 Internal Server Error`

#### Book Events

##### Create Book Event

**Endpoint:** `POST /book_events`

**Request Body:**

```json
{
  "book_id": 123,
  "action": "checkin",  // or "checkout"
  "timestamp": "2006-01-02T15:04:05Z"
}
```

**Response:** `201 Created`

```json
{
  "data": {BookEvent},
  "message": "Book event created"
}
```

**Error Responses:**

- `400 Bad Request`: Invalid JSON or missing required fields
- `500 Internal Server Error`

##### List Book Events

**Endpoint:** `GET /book_events`

**Response:** `200 OK`

```json
{
  "data": [
    {BookEvent},
    {BookEvent},
    ...
  ],
  "message": "Listed book events"
}
```

**Error Responses:**

- `500 Internal Server Error`

### Error Handling

All error responses follow this format:

```json
{
  "message": "Error description",
  "data": {Arbitrary JSON} // optional
}
```

## UI Documentation

### Overview

The front-end of the Simple Library project provides a user-friendly interface for managing books and viewing their check-in/check-out history. It is built using server-side templates and leverages [HTMX](https://htmx.org/) for dynamic interactions without requiring a full page reload.

### Features

- **Book Management**: Add, update, delete, check-in, and check-out books directly from the interface.
- **Dynamic Updates**: Use HTMX to dynamically update parts of the page without reloading.
- **Checkout Log**: View a detailed log of all book check-in and check-out events.

### Routes

The front-end interacts with the server through the following routes:

#### `/` (Root)

**Method**: `GET`
**Description**: Displays the main page with the list of books and the add book form.
**Template**: index.html

#### `/check-out-log`

**Method**: `GET`
**Description**: Displays the checkout log, showing all check-in and check-out events.
**Template**: check_out_log.html

### Error Handling

Error messages are displayed dynamically using HTMX. For example:
- If adding a book fails, an error message is shown below the add book form.
- If an action (e.g., check-in, check-out, delete) fails, an error message is displayed in the appropriate context.

### Styling

The application uses simple CSS styles defined in the base.html template. Key styles include:
- A clean and minimalistic table layout.
- Error messages styled in red for visibility.
- Responsive design for better usability on different screen sizes.

### Future Enhancements

- Add pagination for the book list and checkout log.
- Implement search and filtering for books.
- Enhance error handling with more detailed messages.

## Included Dependencies

Thank you to the talented engineers who provide and maintain their work for free!

- [golang-migrate](https://pkg.go.dev/github.com/golang-migrate/migrate/v4): Database migration creation/runner
- [sqlc](https://github.com/sqlc-dev/sqlc): Generate fully type-safe idiomatic Go code from SQL
- [sqlite](https://pkg.go.dev/modernc.org/sqlite): An in-process implementation of a self-contained, serverless, zero-configuration, transactional SQL database engine
- [Testify](https://pkg.go.dev/github.com/stretchr/testify): Better test assertions
- [uuid](https://pkg.go.dev/github.com/google/uuid): Generates and inspects UUIDs based on [RFC 4122](https://datatracker.ietf.org/doc/html/rfc4122)
