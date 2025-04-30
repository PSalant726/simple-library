package db

import (
	"context"
	"database/sql"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBookEvent(t *testing.T) {
	t.Run("returns the book event with the given ID", func(t *testing.T) {
		ctx := context.Background()
		randomBook := insertRandomBook(t, ctx)
		testBookEvent := insertRandomBookEvent(t, ctx, randomBook.ID)

		bookEvent, err := testQueries.BookEvent(ctx, testBookEvent.ID)
		assert.Equal(t, testBookEvent, bookEvent)
		assert.Nil(t, err)
	})

	t.Run("returns an error", func(t *testing.T) {
		result, err := testQueries.BookEvent(context.Background(), 1_000_000)
		assert.Empty(t, result)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestBookEvents(t *testing.T) {
	t.Run("returns all book events", func(t *testing.T) {
		ctx := context.Background()
		randomBook := insertRandomBook(t, ctx)
		for range 10 {
			insertRandomBookEvent(t, ctx, randomBook.ID)
		}

		bookEvents, err := testQueries.BookEvents(ctx)
		assert.GreaterOrEqual(t, len(bookEvents), 10)
		assert.Nil(t, err)
	})

	t.Run("returns an error", func(t *testing.T) {
		t.Skip()
	})
}

func TestBookEventsByBookID(t *testing.T) {
	t.Run("returns all book events for the provided book ID", func(t *testing.T) {
		ctx := context.Background()
		randomBook := insertRandomBook(t, ctx)
		for range 10 {
			insertRandomBookEvent(t, ctx, randomBook.ID)
		}

		bookEvents, err := testQueries.BookEventsByBookID(ctx, randomBook.ID)
		assert.Len(t, bookEvents, 10)
		assert.Nil(t, err)
	})

	t.Run("returns an error", func(t *testing.T) {
		t.Skip()
	})
}

func TestCreateBookEvent(t *testing.T) {
	t.Run("inserts a book event into the database", func(t *testing.T) {
		ctx := context.Background()
		randomBook := insertRandomBook(t, ctx)
		_ = insertRandomBookEvent(t, ctx, randomBook.ID)
	})

	t.Run("returns an error", func(t *testing.T) {
		ctx := context.Background()
		existingBookEvents, err := testQueries.BookEvents(ctx)
		assert.NotZero(t, len(existingBookEvents))
		assert.Nil(t, err)

		testBookEvent, err := testQueries.CreateBookEvent(ctx, CreateBookEventParams{
			BookID:    1_000_000,
			Action:    "checkin",
			Timestamp: time.Now().UTC(),
		})
		assert.Empty(t, testBookEvent)
		assert.ErrorContains(t, err, "FOREIGN KEY constraint failed")
	})
}

func TestDeleteBookEvent(t *testing.T) {
	t.Run("deletes the book event with the given ID", func(t *testing.T) {
		ctx := context.Background()
		randomBook := insertRandomBook(t, ctx)
		testBookEvent := insertRandomBookEvent(t, ctx, randomBook.ID)

		deletedBookEvent, err := testQueries.DeleteBookEvent(ctx, testBookEvent.ID)
		assert.Equal(t, testBookEvent.ID, deletedBookEvent.ID)
		assert.Nil(t, err)

		result, err := testQueries.BookEvent(ctx, testBookEvent.ID)
		assert.Empty(t, result)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("returns an error", func(t *testing.T) {
		deleted, err := testQueries.DeleteBookEvent(context.Background(), 1_000_000)
		assert.Empty(t, deleted)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestUpdateBookEvent(t *testing.T) {
	t.Run("modifies an existing book event", func(t *testing.T) {
		ctx := context.Background()
		randomBook := insertRandomBook(t, ctx)
		testBookEvent := insertRandomBookEvent(t, ctx, randomBook.ID)

		newBookEventAction := "checkin"
		if testBookEvent.Action == "checkin" {
			newBookEventAction = "checkout"
		}

		updatedBookEvent, err := testQueries.UpdateBookEvent(ctx, UpdateBookEventParams{
			ID:     testBookEvent.ID,
			Action: newBookEventAction,
		})
		assert.Equal(t, testBookEvent.ID, updatedBookEvent.ID)
		assert.Nil(t, err)

		actual, err := testQueries.BookEvent(ctx, testBookEvent.ID)
		assert.Nil(t, err)

		t.Run("fields included in the provided params", func(t *testing.T) {
			assert.NotEqual(t, testBookEvent.Action, actual.Action)
			assert.Equal(t, newBookEventAction, actual.Action)
		})

		t.Run("does not modify fields omitted in the provided params", func(t *testing.T) {
			assert.Equal(t, testBookEvent.ID, actual.ID)
			assert.Equal(t, testBookEvent.BookID, actual.BookID)
			assert.Equal(t, testBookEvent.Timestamp, actual.Timestamp)
		})
	})

	t.Run("returns an error", func(t *testing.T) {
		updatedBookEvent, err := testQueries.UpdateBookEvent(context.Background(), UpdateBookEventParams{
			ID:     1_000_000,
			Action: "checkin",
		})
		assert.Empty(t, updatedBookEvent)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func insertRandomBookEvent(t *testing.T, ctx context.Context, bookID int64) BookEvent {
	t.Helper()

	now := time.Now().UTC()
	actions := []string{"checkin", "checkout"}
	params := CreateBookEventParams{
		BookID:    bookID,
		Action:    actions[rand.IntN(len(actions))],
		Timestamp: now,
	}

	testBookEvent, err := testQueries.CreateBookEvent(ctx, params)
	assert.NotEmpty(t, testBookEvent.ID)
	assert.Equal(t, bookID, testBookEvent.BookID)
	assert.Equal(t, params.Action, testBookEvent.Action)
	assert.Equal(t, params.Timestamp, testBookEvent.Timestamp)
	assert.WithinDuration(t, testBookEvent.CreatedAt, now, time.Second)
	assert.WithinDuration(t, testBookEvent.UpdatedAt, now, time.Second)
	assert.Nil(t, err)

	return testBookEvent
}
