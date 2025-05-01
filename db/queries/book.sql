-- name: CreateBook :one
INSERT INTO books (
        isbn,
        title,
        author,
        description,
        checked_out_at,
        archived_at
    )
VALUES (?1, ?2, ?3, ?4, ?5, ?6)
RETURNING *;
--

-- name: Book :one
SELECT *
FROM books
WHERE id = ?1
LIMIT 1;
--

-- name: Books :many
SELECT *
FROM books
WHERE archived_at IS NULL
ORDER BY id;
--

-- name: UpdateBook :one
UPDATE books
SET isbn = (
        CASE
            WHEN CAST(:isbn AS text) != '' THEN :isbn
            ELSE isbn
        END
    ),
    title = (
        CASE
            WHEN CAST(:title AS text) != '' THEN :title
            ELSE title
        END
    ),
    author = (
        CASE
            WHEN CAST(:author AS text) != '' THEN :author
            ELSE author
        END
    ),
    description = (
        CASE
            WHEN CAST(:description AS text) != '' THEN :description
            ELSE description
        END
    )
WHERE id = ?1
RETURNING *;
--

-- name: CheckOutBook :one
UPDATE books
SET checked_out_at = ?2
WHERE id = ?1
RETURNING *;
--

-- name: CheckInBook :one
UPDATE books
SET checked_out_at = NULL
WHERE id = ?1
RETURNING *;
--

-- name: ArchiveBook :one
UPDATE books
SET archived_at = ?2
WHERE id = ?1
RETURNING *;
--

-- name: DeleteBook :one
DELETE FROM books
WHERE id = ?1
RETURNING *;
