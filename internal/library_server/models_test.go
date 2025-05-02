package library_server

import (
	"testing"

	db "github.com/PSalant726/simple-library/db/sqlc"
	"github.com/stretchr/testify/assert"
)

func TestBook(t *testing.T) {
	t.Run("includes all fields present in the corresponding database model", func(t *testing.T) {
		actual, expected := *new(Book), *new(db.Book)
		assert.Exactly(t, expected.ID, actual.ID)
		assert.Exactly(t, expected.Isbn, actual.Isbn)
		assert.Exactly(t, expected.Title, actual.Title)
		assert.Exactly(t, expected.Author, actual.Author)
		assert.Exactly(t, expected.Description, actual.Description)
		assert.Exactly(t, expected.CheckedOutAt, actual.CheckedOutAt)
		assert.Exactly(t, expected.CreatedAt, actual.CreatedAt)
		assert.Exactly(t, expected.UpdatedAt, actual.UpdatedAt)
		assert.Exactly(t, expected.ArchivedAt, actual.ArchivedAt)
	})
}

func TestUpdateBookRequest(t *testing.T) {
	t.Run("includes all fields necessary to handle an update request", func(t *testing.T) {
		actual := *new(UpdateBookRequest)
		assert.IsType(t, int64(0), actual.ID)
		assert.IsType(t, "", actual.Isbn)
		assert.IsType(t, "", actual.Title)
		assert.IsType(t, "", actual.Author)
		assert.IsType(t, "", actual.Description)
	})
}
