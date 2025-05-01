package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/PSalant726/simple-library/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchiveBook(t *testing.T) {
	t.Run("archives the book with the given ID and time", func(t *testing.T) {
		ctx := context.Background()
		testBook := insertRandomBook(t, ctx)

		archivedBook, err := testQueries.ArchiveBook(ctx, ArchiveBookParams{
			ID: testBook.ID,
			ArchivedAt: sql.NullTime{
				Time:  time.Now().UTC(),
				Valid: true,
			},
		})
		assert.Equal(t, testBook.ID, archivedBook.ID)
		assert.False(t, archivedBook.ArchivedAt.Time.IsZero())
		assert.True(t, archivedBook.ArchivedAt.Valid)
		assert.Nil(t, err)
	})

	t.Run("returns an error", func(t *testing.T) {
		archived, err := testQueries.ArchiveBook(context.Background(), ArchiveBookParams{
			ID: 1_000_000,
			ArchivedAt: sql.NullTime{
				Time:  time.Now().UTC(),
				Valid: true,
			},
		})
		assert.Empty(t, archived)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestBook(t *testing.T) {
	t.Run("returns the book with the given ID", func(t *testing.T) {
		ctx := context.Background()
		testBook := insertRandomBook(t, ctx)

		retrievedBook, err := testQueries.Book(ctx, testBook.ID)
		assert.Equal(t, testBook, retrievedBook)
		assert.Nil(t, err)
	})

	t.Run("returns an error", func(t *testing.T) {
		result, err := testQueries.Book(context.Background(), 1_000_000)
		assert.Empty(t, result)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestBooks(t *testing.T) {
	t.Run("returns all books", func(t *testing.T) {
		ctx := context.Background()
		startingBooks, err := testQueries.Books(ctx)
		require.Nil(t, err)

		for range 10 {
			insertRandomBook(t, ctx)
		}

		books, err := testQueries.Books(ctx)
		assert.Equal(t, len(books), len(startingBooks)+10)
		assert.Nil(t, err)
	})

	t.Run("ignores archived books", func(t *testing.T) {
		ctx := context.Background()
		startingBooks, err := testQueries.Books(ctx)
		require.Nil(t, err)

		archivedBook := insertRandomBook(t, ctx)
		_, err = testQueries.ArchiveBook(ctx, ArchiveBookParams{
			ID: archivedBook.ID,
			ArchivedAt: sql.NullTime{
				Time:  time.Now().UTC(),
				Valid: true,
			},
		})
		require.Nil(t, err)

		books, err := testQueries.Books(ctx)
		assert.Equal(t, len(books), len(startingBooks))
		assert.Nil(t, err)
	})

	t.Run("returns an error", func(t *testing.T) {
		t.Skip()
	})
}

func TestCheckInBook(t *testing.T) {
	t.Run("checks in the book with the given ID", func(t *testing.T) {
		ctx := context.Background()
		testBook := insertRandomBook(t, ctx)

		checkedOutBook, err := testQueries.CheckOutBook(ctx, CheckOutBookParams{
			ID: testBook.ID,
			CheckedOutAt: sql.NullTime{
				Time:  time.Now().UTC(),
				Valid: true,
			},
		})
		require.Equal(t, testBook.ID, checkedOutBook.ID)
		require.False(t, checkedOutBook.CheckedOutAt.Time.IsZero())
		require.True(t, checkedOutBook.CheckedOutAt.Valid)
		require.Nil(t, err)

		checkedInBook, err := testQueries.CheckInBook(ctx, checkedOutBook.ID)
		assert.Equal(t, testBook.ID, checkedInBook.ID)
		assert.True(t, checkedInBook.CheckedOutAt.Time.IsZero())
		assert.False(t, checkedInBook.CheckedOutAt.Valid)
		assert.Nil(t, err)
	})

	t.Run("returns an error", func(t *testing.T) {
		checkedInBook, err := testQueries.CheckInBook(context.Background(), 1_000_000)
		assert.Empty(t, checkedInBook)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestCheckOutBook(t *testing.T) {
	t.Run("checks out the book with the given ID", func(t *testing.T) {
		ctx := context.Background()
		testBook := insertRandomBook(t, ctx)

		checkedOutBook, err := testQueries.CheckOutBook(ctx, CheckOutBookParams{
			ID: testBook.ID,
			CheckedOutAt: sql.NullTime{
				Time:  time.Now().UTC(),
				Valid: true,
			},
		})
		assert.Equal(t, testBook.ID, checkedOutBook.ID)
		assert.False(t, checkedOutBook.CheckedOutAt.Time.IsZero())
		assert.True(t, checkedOutBook.CheckedOutAt.Valid)
		assert.Nil(t, err)
	})

	t.Run("returns an error", func(t *testing.T) {
		checkedOutBook, err := testQueries.CheckOutBook(context.Background(), CheckOutBookParams{
			ID: 1_000_000,
			CheckedOutAt: sql.NullTime{
				Time:  time.Now().UTC(),
				Valid: true,
			},
		})
		assert.Empty(t, checkedOutBook)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestCreateBook(t *testing.T) {
	t.Run("inserts a book into the database", func(t *testing.T) {
		_ = insertRandomBook(t, context.Background())
	})

	t.Run("returns an error", func(t *testing.T) {
		ctx := context.Background()
		existingBooks, err := testQueries.Books(ctx)
		require.NotZero(t, len(existingBooks))
		require.Nil(t, err)

		testBook, err := testQueries.CreateBook(ctx, CreateBookParams{
			Isbn:   existingBooks[0].Isbn,
			Title:  util.RandomString(10),
			Author: util.RandomString(10),
			Description: sql.NullString{
				String: util.RandomString(30),
				Valid:  true,
			},
		})
		assert.Empty(t, testBook)
		assert.ErrorContains(t, err, "UNIQUE constraint failed")
	})
}

func TestDeleteBook(t *testing.T) {
	t.Run("deletes the book with the given ID", func(t *testing.T) {
		ctx := context.Background()
		testBook := insertRandomBook(t, ctx)

		deletedBook, err := testQueries.DeleteBook(ctx, testBook.ID)
		assert.Equal(t, testBook.ID, deletedBook.ID)
		assert.Nil(t, err)

		result, err := testQueries.Book(ctx, testBook.ID)
		assert.Empty(t, result)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("returns an error", func(t *testing.T) {
		deleted, err := testQueries.DeleteBook(context.Background(), 1_000_000)
		assert.Empty(t, deleted)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestUpdateBook(t *testing.T) {
	t.Run("modifies an existing book", func(t *testing.T) {
		ctx := context.Background()
		testBook := insertRandomBook(t, ctx)

		time.Sleep(time.Second)
		updatedBook, err := testQueries.UpdateBook(ctx, UpdateBookParams{
			ID:     testBook.ID,
			Author: "phil",
		})
		assert.Equal(t, testBook.ID, updatedBook.ID)
		assert.Nil(t, err)

		actual, err := testQueries.Book(ctx, testBook.ID)
		assert.Nil(t, err)

		t.Run("fields included in the provided params", func(t *testing.T) {
			assert.NotEqual(t, testBook.Author, actual.Author)
			assert.Equal(t, "phil", actual.Author)
		})

		t.Run("the updated_at field", func(t *testing.T) {
			assert.True(t, actual.UpdatedAt.After(testBook.UpdatedAt))
		})

		t.Run("does not modify fields omitted in the provided params", func(t *testing.T) {
			assert.Equal(t, testBook.Isbn, actual.Isbn)
			assert.Equal(t, testBook.Title, actual.Title)
			assert.Equal(t, testBook.Description, actual.Description)
			assert.Equal(t, testBook.CreatedAt, actual.CreatedAt)
		})
	})

	t.Run("returns an error", func(t *testing.T) {
		updatedBook, err := testQueries.UpdateBook(context.Background(), UpdateBookParams{
			ID:     1_000_000,
			Author: "phil",
		})
		assert.Empty(t, updatedBook)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func insertRandomBook(t *testing.T, ctx context.Context) Book {
	t.Helper()

	params := CreateBookParams{
		Isbn:   util.RandomString(10),
		Title:  util.RandomString(10),
		Author: util.RandomString(10),
		Description: sql.NullString{
			String: util.RandomString(30),
			Valid:  true,
		},
		CheckedOutAt: sql.NullTime{},
		ArchivedAt:   sql.NullTime{},
	}

	now := time.Now().UTC()
	testBook, err := testQueries.CreateBook(ctx, params)
	assert.NotEmpty(t, testBook.ID)
	assert.Equal(t, params.Isbn, testBook.Isbn)
	assert.Equal(t, params.Title, testBook.Title)
	assert.Equal(t, params.Author, testBook.Author)
	assert.Equal(t, params.Description.String, testBook.Description.String)
	assert.True(t, testBook.Description.Valid)
	assert.True(t, testBook.CheckedOutAt.Time.IsZero())
	assert.False(t, testBook.CheckedOutAt.Valid)
	assert.WithinDuration(t, testBook.CreatedAt, now, time.Second)
	assert.WithinDuration(t, testBook.UpdatedAt, now, time.Second)
	assert.True(t, testBook.ArchivedAt.Time.IsZero())
	assert.False(t, testBook.ArchivedAt.Valid)
	assert.Nil(t, err)

	return testBook
}
