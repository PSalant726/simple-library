-- name: CreateBookEvent :one
INSERT INTO book_events (book_id, action, timestamp)
VALUES (?1, ?2, ?3)
RETURNING *;
--

-- name: BookEvent :one
SELECT *
FROM book_events
WHERE id = ?1
LIMIT 1;
--

-- name: BookEvents :many
SELECT *
FROM book_events
ORDER BY id;
--

-- name: BookEventsByBookID :many
SELECT *
FROM book_events
WHERE book_id = ?1
ORDER BY timestamp DESC;
--

-- name: UpdateBookEvent :one
UPDATE book_events
SET book_id = CASE
        WHEN CAST(:book_id AS integer) != 0 THEN :book_id
        ELSE book_id
    END,
    action = CASE
        WHEN CAST(:action AS text) != '' THEN :action
        ELSE action
    END,
    timestamp = CASE
        WHEN CAST(:should_update_timestamp AS bool) THEN ?2
        ELSE timestamp
    END
WHERE id = ?1
RETURNING *;
--

-- name: DeleteBookEvent :one
DELETE FROM book_events
WHERE id = ?1
RETURNING *;
--
