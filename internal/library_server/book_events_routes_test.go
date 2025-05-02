package library_server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	db "github.com/PSalant726/simple-library/db/sqlc"
	"github.com/PSalant726/simple-library/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_handleCreateBookEvent(t *testing.T) {
	t.Run("with a valid request", func(t *testing.T) {
		now := time.Now().UTC()
		testBookEvent := BookEvent{
			BookID:    1,
			Action:    bookActionCheckIn,
			Timestamp: now,
		}

		body, err := json.Marshal(testBookEvent)
		require.Nil(t, err)

		req := httptest.NewRequest(http.MethodPost, routeBookEvents, bytes.NewBuffer(body))
		req.Header.Set(headerKeyContentType, headerValueApplicationJSON)
		w := httptest.NewRecorder()

		var beforeBookEvents, afterBookEvents []db.BookEvent
		withStartedTestServer(t, func(s *Server) {
			var err error

			ctx := context.Background()
			_, err = s.libraryDB.CreateBook(ctx, db.CreateBookParams{
				Isbn:   util.RandomString(10),
				Title:  util.RandomString(10),
				Author: util.RandomString(10),
			})
			require.Nil(t, err)

			beforeBookEvents, err = s.libraryDB.BookEvents(ctx)
			require.Nil(t, err)

			s.handleCreateBookEvent(w, req)

			afterBookEvents, err = s.libraryDB.BookEvents(ctx)
			require.Nil(t, err)
		})

		t.Run("responds with the expected status code", func(t *testing.T) {
			assert.Equal(t, http.StatusCreated, w.Code)
		})

		t.Run("responds with valid JSON", func(t *testing.T) {
			var resp Response
			assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

			t.Run("with the expected content", func(t *testing.T) {
				assert.Equal(t, "Book event created.", resp.Message)

				var respBookEvent BookEvent
				require.Nil(t, json.Unmarshal(resp.Data, &respBookEvent))

				assert.Equal(t, testBookEvent.BookID, respBookEvent.BookID)
				assert.Equal(t, testBookEvent.Action, respBookEvent.Action)
				assert.Equal(t, testBookEvent.Timestamp, respBookEvent.Timestamp)
			})
		})

		t.Run("inserts a new book into the database", func(t *testing.T) {
			assert.Len(t, afterBookEvents, len(beforeBookEvents)+1)
		})
	})

	t.Run("with an invalid request", func(t *testing.T) {
		tests := []struct {
			description        string
			reqBody            *bytes.Buffer
			expectedStatusCode int
			expectedMessage    string
		}{
			{
				description:        "with an empty request body",
				reqBody:            &bytes.Buffer{},
				expectedStatusCode: http.StatusBadRequest,
				expectedMessage:    "empty request body",
			},
			{
				description:        "with an invalid request body",
				reqBody:            bytes.NewBufferString(`{"foo:"bar"}`),
				expectedStatusCode: http.StatusBadRequest,
				expectedMessage:    "failed to parse JSON: invalid character 'b' after object key",
			},
			{
				description: "with a request body exceeding the maximum size",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"book_id": %d, "action": %q, "timestamp": %q}`,
					1,
					util.RandomString(10_000_000),
					time.Now().UTC().Format(time.RFC3339),
				)),
				expectedStatusCode: http.StatusRequestEntityTooLarge,
				expectedMessage:    "http: request body too large",
			},
			{
				description: "with a zero book ID",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"book_id": %d, "action": %q, "timestamp": %q}`,
					0,
					bookActionCheckIn,
					time.Now().UTC().Format(time.RFC3339),
				)),
				expectedStatusCode: http.StatusBadRequest,
				expectedMessage:    "Missing or invalid field(s).",
			},
			{
				description: "with an invalid book action",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"book_id": %d, "action": %q, "timestamp": %q}`,
					1,
					util.RandomString(10),
					time.Now().UTC().Format(time.RFC3339),
				)),
				expectedStatusCode: http.StatusBadRequest,
				expectedMessage:    "Missing or invalid field(s).",
			},
			{
				description: "with a missing (zero) timestamp",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"book_id": %d, "action": %q, "timestamp": %q}`,
					1,
					bookActionCheckIn,
					time.Time{}.Format(time.RFC3339),
				)),
				expectedStatusCode: http.StatusBadRequest,
				expectedMessage:    "Missing or invalid field(s).",
			},
		}

		for _, test := range tests {
			t.Run(test.description, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, routeBookEvents, test.reqBody)
				req.Header.Set(headerKeyContentType, headerValueApplicationJSON)
				w := httptest.NewRecorder()

				var beforeBookEvents, afterBookEvents []db.BookEvent
				withStartedTestServer(t, func(s *Server) {
					var err error

					ctx := context.Background()

					_, err = s.libraryDB.CreateBook(ctx, db.CreateBookParams{
						Isbn:   util.RandomString(10),
						Title:  util.RandomString(10),
						Author: util.RandomString(10),
					})
					require.Nil(t, err)

					beforeBookEvents, err = s.libraryDB.BookEvents(ctx)
					require.Nil(t, err)

					s.handleCreateBookEvent(w, req)

					afterBookEvents, err = s.libraryDB.BookEvents(ctx)
					require.Nil(t, err)
				})

				t.Run("responds with the expected status code", func(t *testing.T) {
					assert.Equal(t, test.expectedStatusCode, w.Code)
				})

				t.Run("responds with valid JSON", func(t *testing.T) {
					var resp Response
					assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

					t.Run("with the expected message", func(t *testing.T) {
						assert.Equal(t, test.expectedMessage, resp.Message)
					})
				})

				t.Run("does not insert a new book event into the database", func(t *testing.T) {
					assert.Equal(t, beforeBookEvents, afterBookEvents)
				})
			})
		}
	})

	t.Run("when the database insert fails", func(t *testing.T) {
		t.Skip()
	})
}

func TestServer_handleDeleteBookEvent(t *testing.T) {
	tests := []struct {
		description        string
		bookEventID        string
		expectedMessage    string
		expectedStatusCode int
	}{
		{
			description:        "with an existing book event ID",
			bookEventID:        "1",
			expectedMessage:    "Deleted book event with ID 1.",
			expectedStatusCode: http.StatusOK,
		},
		{
			description:        "with a book event ID that does not exist",
			bookEventID:        "1000000",
			expectedMessage:    "Book event with ID 1000000 not found.",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			description:        "with an invalid book event ID value",
			bookEventID:        "foo",
			expectedMessage:    "Invalid book event ID. Must be an integer.",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodDelete,
				fmt.Sprintf("%s/%s", routeBookEvents, test.bookEventID),
				nil,
			)
			req.SetPathValue("id", test.bookEventID)
			w := httptest.NewRecorder()

			withStartedTestServer(t, func(s *Server) {
				ctx := context.Background()

				book, err := s.libraryDB.CreateBook(ctx, db.CreateBookParams{
					Isbn:   util.RandomString(10),
					Title:  util.RandomString(10),
					Author: util.RandomString(10),
				})
				require.Nil(t, err)

				_, err = s.libraryDB.CreateBookEvent(ctx, db.CreateBookEventParams{
					BookID:    book.ID,
					Action:    bookActionCheckIn,
					Timestamp: time.Now().UTC(),
				})
				require.Nil(t, err)

				s.handleDeleteBookEvent(w, req)
			})

			t.Run("responds with the expected status code", func(t *testing.T) {
				assert.Equal(t, test.expectedStatusCode, w.Code)
			})

			t.Run("responds with valid JSON", func(t *testing.T) {
				var resp Response
				assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

				t.Run("with the expected content", func(t *testing.T) {
					assert.Equal(t, test.expectedMessage, resp.Message)
					assert.Empty(t, resp.Data)
				})
			})
		})
	}

	t.Run("when the database query fails", func(t *testing.T) {
		t.Skip()
	})
}

func TestServer_handleGetBookEvent(t *testing.T) {
	tests := []struct {
		description        string
		bookEventID        string
		expectedData       bool
		expectedMessage    string
		expectedStatusCode int
	}{
		{
			description:        "with an existing book event ID",
			bookEventID:        "1",
			expectedData:       true,
			expectedMessage:    "Book event found.",
			expectedStatusCode: http.StatusOK,
		},
		{
			description:        "with an book event ID that does not exist",
			bookEventID:        "1000000",
			expectedData:       false,
			expectedMessage:    "Book event with ID 1000000 not found.",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			description:        "with an invalid book event ID value",
			bookEventID:        "foo",
			expectedData:       false,
			expectedMessage:    "Invalid book event ID. Must be an integer.",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("%s/%s", routeBookEvents, test.bookEventID),
				nil,
			)
			req.SetPathValue("id", test.bookEventID)
			w := httptest.NewRecorder()

			withStartedTestServer(t, func(s *Server) {
				ctx := context.Background()
				book, err := s.libraryDB.CreateBook(ctx, db.CreateBookParams{
					Isbn:   util.RandomString(10),
					Title:  util.RandomString(10),
					Author: util.RandomString(10),
				})
				require.Nil(t, err)

				_, err = s.libraryDB.CreateBookEvent(ctx, db.CreateBookEventParams{
					BookID:    book.ID,
					Action:    bookActionCheckIn,
					Timestamp: time.Now().UTC(),
				})
				require.Nil(t, err)

				s.handleGetBookEvent(w, req)
			})

			t.Run("responds with the expected status code", func(t *testing.T) {
				assert.Equal(t, test.expectedStatusCode, w.Code)
			})

			t.Run("responds with valid JSON", func(t *testing.T) {
				var resp Response
				assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

				t.Run("with the expected content", func(t *testing.T) {
					assert.Equal(t, test.expectedMessage, resp.Message)

					if test.expectedData {
						var respBookEvent BookEvent
						require.Nil(t, json.Unmarshal(resp.Data, &respBookEvent))

						assert.NotEmpty(t, respBookEvent)
					} else {
						assert.Empty(t, resp.Data)
					}
				})
			})
		})
	}

	t.Run("when the database query fails", func(t *testing.T) {
		t.Skip()
	})

	t.Run("when marshaling the response JSON fails", func(t *testing.T) {
		t.Skip()
	})
}

func TestServer_handleListBookEvents(t *testing.T) {
	t.Run("with a valid request", func(t *testing.T) {
		tests := []struct {
			description     string
			expectedResults int
			testBookEvents  []BookEvent
		}{
			{
				description:     "with multiple book events",
				expectedResults: 2,
				testBookEvents: []BookEvent{
					{
						BookID:    1,
						Action:    bookActionCheckIn,
						Timestamp: time.Now().UTC(),
					},
					{
						BookID:    1,
						Action:    bookActionCheckIn,
						Timestamp: time.Now().UTC().Add(time.Hour),
					},
				},
			},
			{
				description:     "with no book events found",
				expectedResults: 0,
				testBookEvents:  []BookEvent{},
			},
		}

		for _, test := range tests {
			req := httptest.NewRequest(http.MethodGet, routeBookEvents, nil)
			w := httptest.NewRecorder()

			withStartedTestServer(t, func(s *Server) {
				ctx := context.Background()
				_, err := s.libraryDB.CreateBook(ctx, db.CreateBookParams{
					Isbn:   util.RandomString(10),
					Title:  util.RandomString(10),
					Author: util.RandomString(10),
				})
				require.Nil(t, err)

				for _, bookEvent := range test.testBookEvents {
					_, err := s.libraryDB.CreateBookEvent(ctx, db.CreateBookEventParams{
						BookID:    bookEvent.BookID,
						Action:    bookEvent.Action,
						Timestamp: bookEvent.Timestamp,
					})
					require.Nil(t, err)

				}

				s.handleListBookEvents(w, req)
			})

			t.Run(test.description, func(t *testing.T) {
				t.Run("responds with the expected status code", func(t *testing.T) {
					assert.Equal(t, http.StatusOK, w.Code)
				})

				t.Run("responds with valid JSON", func(t *testing.T) {
					var resp Response
					assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

					t.Run("with the expected content", func(t *testing.T) {
						assert.Equal(t, "Listed book events.", resp.Message)

						var respBooks []BookEvent
						require.Nil(t, json.Unmarshal(resp.Data, &respBooks))

						assert.Len(t, respBooks, test.expectedResults)
					})
				})
			})
		}
	})

	t.Run("when the database query fails", func(t *testing.T) {
		t.Skip()
	})

	t.Run("when marshaling the response JSON fails", func(t *testing.T) {
		t.Skip()
	})
}

func TestServer_handleListBookEventsByBookID(t *testing.T) {
	t.Run("with a valid request", func(t *testing.T) {
		tests := []struct {
			description     string
			bookID          string
			expectedResults int
			testBookEvents  []BookEvent
		}{
			{
				description:     "with no book events",
				bookID:          "1",
				expectedResults: 0,
				testBookEvents:  []BookEvent{},
			},
			{
				description:     "with multiple book events",
				bookID:          "1",
				expectedResults: 2,
				testBookEvents: []BookEvent{
					{
						BookID:    1,
						Action:    bookActionCheckIn,
						Timestamp: time.Now().UTC(),
					},
					{
						BookID:    1,
						Action:    bookActionCheckIn,
						Timestamp: time.Now().UTC().Add(time.Hour),
					},
				},
			},
			{
				description:     "with an existing book ID and no book events found",
				bookID:          "2",
				expectedResults: 0,
				testBookEvents: []BookEvent{
					{
						BookID:    1,
						Action:    bookActionCheckIn,
						Timestamp: time.Now().UTC(),
					},
					{
						BookID:    1,
						Action:    bookActionCheckIn,
						Timestamp: time.Now().UTC().Add(time.Hour),
					},
				},
			},
			{
				description:     "with a book ID that does not exist",
				bookID:          "1000",
				expectedResults: 0,
				testBookEvents: []BookEvent{
					{
						BookID:    1,
						Action:    bookActionCheckIn,
						Timestamp: time.Now().UTC(),
					},
					{
						BookID:    1,
						Action:    bookActionCheckIn,
						Timestamp: time.Now().UTC().Add(time.Hour),
					},
				},
			},
		}

		for _, test := range tests {
			req := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("%s/book/%s", routeBookEvents, test.bookID),
				nil,
			)
			req.SetPathValue("id", test.bookID)
			w := httptest.NewRecorder()

			withStartedTestServer(t, func(s *Server) {
				ctx := context.Background()
				for range 2 {
					_, err := s.libraryDB.CreateBook(ctx, db.CreateBookParams{
						Isbn:   util.RandomString(10),
						Title:  util.RandomString(10),
						Author: util.RandomString(10),
					})
					require.Nil(t, err)
				}

				for _, bookEvent := range test.testBookEvents {
					_, err := s.libraryDB.CreateBookEvent(ctx, db.CreateBookEventParams{
						BookID:    bookEvent.BookID,
						Action:    bookEvent.Action,
						Timestamp: bookEvent.Timestamp,
					})
					require.Nil(t, err)

				}

				s.handleListBookEventsByBookID(w, req)
			})

			t.Run(test.description, func(t *testing.T) {
				t.Run("responds with the expected status code", func(t *testing.T) {
					assert.Equal(t, http.StatusOK, w.Code)
				})

				t.Run("responds with valid JSON", func(t *testing.T) {
					var resp Response
					assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

					t.Run("with the expected content", func(t *testing.T) {
						assert.Equal(
							t,
							fmt.Sprintf("Listed book events with book ID %s.", test.bookID),
							resp.Message,
						)

						var respBooks []BookEvent
						require.Nil(t, json.Unmarshal(resp.Data, &respBooks))

						assert.Len(t, respBooks, test.expectedResults)
					})
				})
			})
		}
	})

	t.Run("with an invalid request", func(t *testing.T) {
		t.Run("with an invalid book ID value", func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("%s/book/foo", routeBookEvents),
				nil,
			)
			req.SetPathValue("id", "foo")
			w := httptest.NewRecorder()

			withStartedTestServer(t, func(s *Server) {
				ctx := context.Background()
				book, err := s.libraryDB.CreateBook(ctx, db.CreateBookParams{
					Isbn:   util.RandomString(10),
					Title:  util.RandomString(10),
					Author: util.RandomString(10),
				})
				require.Nil(t, err)

				_, err = s.libraryDB.CreateBookEvent(ctx, db.CreateBookEventParams{
					BookID:    book.ID,
					Action:    bookActionCheckIn,
					Timestamp: time.Now().UTC(),
				})
				require.Nil(t, err)

				s.handleListBookEventsByBookID(w, req)
			})

			t.Run("responds with the expected status code", func(t *testing.T) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			})

			t.Run("responds with valid JSON", func(t *testing.T) {
				var resp Response
				assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

				t.Run("with the expected content", func(t *testing.T) {
					assert.Equal(t, "Invalid book ID. Must be an integer.", resp.Message)
					assert.Empty(t, resp.Data)
				})
			})

		})
	})

	t.Run("when the database query fails", func(t *testing.T) {
		t.Skip()
	})

	t.Run("when marshaling the response JSON fails", func(t *testing.T) {
		t.Skip()
	})
}

func TestServer_handleUpdateBookEvent(t *testing.T) {
	t.Run("with a valid request", func(t *testing.T) {
		testBody := UpdateBookEventRequest{
			ID:        1,
			BookID:    2,
			Action:    bookActionCheckOut,
			Timestamp: time.Now().UTC().Add(time.Hour),
		}

		body, err := json.Marshal(testBody)
		require.Nil(t, err)

		req := httptest.NewRequest(http.MethodPut, routeBookEvents, bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		withStartedTestServer(t, func(s *Server) {
			ctx := context.Background()
			for range 2 {
				_, err := s.libraryDB.CreateBook(ctx, db.CreateBookParams{
					Isbn:   util.RandomString(10),
					Title:  util.RandomString(10),
					Author: util.RandomString(10),
				})
				require.Nil(t, err)
			}

			_, err = s.libraryDB.CreateBookEvent(ctx, db.CreateBookEventParams{
				BookID:    1,
				Action:    bookActionCheckIn,
				Timestamp: time.Now().UTC(),
			})
			require.Nil(t, err)

			s.handleUpdateBookEvent(w, req)
		})

		t.Run("responds with the expected status code", func(t *testing.T) {
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("responds with valid JSON", func(t *testing.T) {
			var resp Response
			assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

			t.Run("with the expected content", func(t *testing.T) {
				assert.Equal(t, "Book event updated.", resp.Message)

				var respBookEvent BookEvent
				require.Nil(t, json.Unmarshal(resp.Data, &respBookEvent))

				assert.NotEmpty(t, respBookEvent)
				assert.Equal(t, testBody.ID, respBookEvent.ID)
				assert.Equal(t, testBody.BookID, respBookEvent.BookID)
				assert.Equal(t, testBody.Action, respBookEvent.Action)
				assert.Equal(t, testBody.Timestamp, respBookEvent.Timestamp)
			})
		})
	})

	t.Run("with an invalid request", func(t *testing.T) {
		tests := []struct {
			description        string
			reqBody            *bytes.Buffer
			expectedMessage    string
			expectedStatusCode int
		}{
			{
				description: "with a zero book event ID",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"id": 0, "book_id": %d, "action": %q}`,
					1,
					bookActionCheckOut,
				)),
				expectedMessage:    "Missing or invalid field(s).",
				expectedStatusCode: http.StatusBadRequest,
			},
			{
				description: "with a book event ID that does not exist",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"id": 1000000, "book_id": %d, "action": %q}`,
					1,
					bookActionCheckOut,
				)),
				expectedMessage:    "Book event with ID 1000000 not found.",
				expectedStatusCode: http.StatusNotFound,
			},
			{
				description:        "with an empty request body",
				reqBody:            &bytes.Buffer{},
				expectedStatusCode: http.StatusBadRequest,
				expectedMessage:    "empty request body",
			},
			{
				description:        "with an invalid request body",
				reqBody:            bytes.NewBufferString(`{"id": 1, "foo: "bar"}`),
				expectedStatusCode: http.StatusBadRequest,
				expectedMessage:    "failed to parse JSON: invalid character 'b' after object key",
			},
			{
				description: "with a request body exceeding the maximum size",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"id": 1, "book_id": %d, "action": %q}`,
					1,
					util.RandomString(10_000_000),
				)),
				expectedStatusCode: http.StatusRequestEntityTooLarge,
				expectedMessage:    "http: request body too large",
			},
			{
				description: "with a zero book ID",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"id": 1, "book_id": %d, "action": %q}`,
					0,
					bookActionCheckOut,
				)),
				expectedStatusCode: http.StatusBadRequest,
				expectedMessage:    "Missing or invalid field(s).",
			},
			{
				description: "with a book ID that does not exist",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"id": 1, "book_id": %d, "action": %q}`,
					1000,
					bookActionCheckOut,
				)),
				expectedStatusCode: http.StatusBadRequest,
				expectedMessage:    "Book with ID 1000 not found.",
			},
			{
				description: "with an invalid book action value",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"id": 1, "book_id": %d, "action": %q}`,
					1,
					util.RandomString(10),
				)),
				expectedStatusCode: http.StatusBadRequest,
				expectedMessage:    "Missing or invalid field(s).",
			},
		}

		for _, test := range tests {
			t.Run(test.description, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPut, routeBookEvents, test.reqBody)
				w := httptest.NewRecorder()

				withStartedTestServer(t, func(s *Server) {
					ctx := context.Background()
					_, err := s.libraryDB.CreateBook(ctx, db.CreateBookParams{
						Isbn:   util.RandomString(10),
						Title:  util.RandomString(10),
						Author: util.RandomString(10),
					})
					require.Nil(t, err)

					_, err = s.libraryDB.CreateBookEvent(ctx, db.CreateBookEventParams{
						BookID:    1,
						Action:    bookActionCheckIn,
						Timestamp: time.Now().UTC(),
					})
					require.Nil(t, err)

					s.handleUpdateBookEvent(w, req)
				})

				t.Run("responds with the expected status code", func(t *testing.T) {
					assert.Equal(t, test.expectedStatusCode, w.Code)
				})

				t.Run("responds with valid JSON", func(t *testing.T) {
					var resp Response
					assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

					t.Run("with the expected content", func(t *testing.T) {
						assert.Equal(t, test.expectedMessage, resp.Message)
					})
				})
			})
		}
	})

	t.Run("with no timestamp provided", func(t *testing.T) {
		testBody := UpdateBookEventRequest{
			ID:     1,
			BookID: 1,
			Action: bookActionCheckOut,
		}

		body, err := json.Marshal(testBody)
		require.Nil(t, err)

		req := httptest.NewRequest(http.MethodPut, routeBookEvents, bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		now := time.Now().UTC()
		withStartedTestServer(t, func(s *Server) {
			ctx := context.Background()
			_, err := s.libraryDB.CreateBook(ctx, db.CreateBookParams{
				Isbn:   util.RandomString(10),
				Title:  util.RandomString(10),
				Author: util.RandomString(10),
			})
			require.Nil(t, err)

			_, err = s.libraryDB.CreateBookEvent(ctx, db.CreateBookEventParams{
				BookID:    1,
				Action:    bookActionCheckIn,
				Timestamp: now,
			})
			require.Nil(t, err)

			s.handleUpdateBookEvent(w, req)
		})

		t.Run("responds with the expected status code", func(t *testing.T) {
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("responds with valid JSON", func(t *testing.T) {
			var resp Response
			assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

			t.Run("with the expected content", func(t *testing.T) {
				assert.Equal(t, "Book event updated.", resp.Message)

				var respBookEvent BookEvent
				require.Nil(t, json.Unmarshal(resp.Data, &respBookEvent))

				assert.NotEmpty(t, respBookEvent)
				assert.Equal(t, testBody.ID, respBookEvent.ID)
				assert.Equal(t, testBody.BookID, respBookEvent.BookID)
				assert.Equal(t, testBody.Action, respBookEvent.Action)
				assert.Equal(t, now, respBookEvent.Timestamp)
			})
		})
	})

	t.Run("when the database query fails", func(t *testing.T) {
		t.Skip()
	})

	t.Run("when marshaling the response JSON fails", func(t *testing.T) {
		t.Skip()
	})
}
