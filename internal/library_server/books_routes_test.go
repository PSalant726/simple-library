package library_server

import (
	"bytes"
	"context"
	"database/sql"
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

func TestServer_handleArchiveBook(t *testing.T) {
	t.Run("with a valid request", func(t *testing.T) {
		testBody := ArchiveBookRequest{
			ID:         1,
			ArchivedAt: time.Now().UTC(),
		}

		body, err := json.Marshal(testBody)
		require.Nil(t, err)

		req := httptest.NewRequest(
			http.MethodPatch,
			fmt.Sprintf("%s/archive", routeBooks),
			bytes.NewBuffer(body),
		)
		w := httptest.NewRecorder()

		withStartedTestServer(t, func(s *Server) {
			body, err := json.Marshal(Book{
				Isbn:   util.RandomString(10),
				Title:  util.RandomString(10),
				Author: util.RandomString(10),
			})
			require.Nil(t, err)

			s.handleCreateBook(
				httptest.NewRecorder(),
				httptest.NewRequest(http.MethodPost, routeBooks, bytes.NewBuffer(body)),
			)

			s.handleArchiveBook(w, req)
		})

		t.Run("responds with the expected status code", func(t *testing.T) {
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("responds with valid JSON", func(t *testing.T) {
			var resp Response
			assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

			t.Run("with the expected content", func(t *testing.T) {
				assert.Equal(t, "Book archived.", resp.Message)

				var respBook Book
				require.Nil(t, json.Unmarshal(resp.Data, &respBook))

				assert.NotEmpty(t, respBook)
				assert.Equal(t, int64(1), respBook.ID)
				assert.Equal(t, testBody.ArchivedAt, respBook.ArchivedAt.Time)
			})
		})
	})

	t.Run("with an invalid request", func(t *testing.T) {
		tests := []struct {
			description        string
			archiveTimestamp   time.Time
			bookID             int64
			expectedMessage    string
			expectedStatusCode int
		}{
			{
				description:        "with a book ID that does not exist",
				archiveTimestamp:   time.Now().UTC(),
				bookID:             1000000,
				expectedMessage:    "Book with ID 1000000 not found.",
				expectedStatusCode: http.StatusNotFound,
			},
			{
				description:        "with a missing/zero archive timestamp",
				archiveTimestamp:   time.Time{},
				bookID:             1,
				expectedMessage:    "Archive timestamp not provided.",
				expectedStatusCode: http.StatusBadRequest,
			},
		}

		for _, test := range tests {
			t.Run(test.description, func(t *testing.T) {
				body, err := json.Marshal(ArchiveBookRequest{
					ID:         test.bookID,
					ArchivedAt: test.archiveTimestamp,
				})
				require.Nil(t, err)

				req := httptest.NewRequest(
					http.MethodPatch,
					fmt.Sprintf("%s/archive", routeBooks),
					bytes.NewBuffer(body),
				)
				w := httptest.NewRecorder()

				withStartedTestServer(t, func(s *Server) {
					body, err := json.Marshal(Book{
						Isbn:   util.RandomString(10),
						Title:  util.RandomString(10),
						Author: util.RandomString(10),
					})
					require.Nil(t, err)

					s.handleCreateBook(
						httptest.NewRecorder(),
						httptest.NewRequest(http.MethodPost, routeBooks, bytes.NewBuffer(body)),
					)

					s.handleArchiveBook(w, req)
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
	})

	t.Run("when the database query fails", func(t *testing.T) {
		t.Skip()
	})

	t.Run("when marshaling the response JSON fails", func(t *testing.T) {
		t.Skip()
	})
}

func TestServer_handleCreateBook(t *testing.T) {
	t.Run("with a valid request", func(t *testing.T) {
		testBook := Book{
			Isbn:   util.RandomString(10),
			Title:  util.RandomString(10),
			Author: util.RandomString(10),
		}

		body, err := json.Marshal(testBook)
		require.Nil(t, err)

		req := httptest.NewRequest(http.MethodPost, routeBooks, bytes.NewBuffer(body))
		req.Header.Set(headerKeyContentType, headerValueApplicationJSON)
		w := httptest.NewRecorder()

		var beforeBooks, afterBooks []db.Book
		withStartedTestServer(t, func(s *Server) {
			var err error

			ctx := context.Background()
			beforeBooks, err = s.libraryDB.Books(ctx)
			require.Nil(t, err)

			s.handleCreateBook(w, req)

			afterBooks, err = s.libraryDB.Books(ctx)
			require.Nil(t, err)
		})

		t.Run("responds with the expected status code", func(t *testing.T) {
			assert.Equal(t, http.StatusCreated, w.Code)
		})

		t.Run("responds with valid JSON", func(t *testing.T) {
			var resp Response
			assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

			t.Run("with the expected content", func(t *testing.T) {
				assert.Equal(t, "Book created.", resp.Message)

				var respBook Book
				require.Nil(t, json.Unmarshal(resp.Data, &respBook))

				assert.Equal(t, testBook.Isbn, respBook.Isbn)
				assert.Equal(t, testBook.Title, respBook.Title)
				assert.Equal(t, testBook.Author, respBook.Author)
			})
		})

		t.Run("inserts a new book into the database", func(t *testing.T) {
			assert.Len(t, afterBooks, len(beforeBooks)+1)
		})
	})

	t.Run("with an invalid request", func(t *testing.T) {
		tests := []struct {
			description string
			reqBody     *bytes.Buffer
			statusCode  int
			message     string
		}{
			{
				description: "with an empty request body",
				reqBody:     &bytes.Buffer{},
				statusCode:  http.StatusBadRequest,
				message:     "empty request body",
			},
			{
				description: "with an invalid request body",
				reqBody:     bytes.NewBufferString(`{"foo:"bar"}`),
				statusCode:  http.StatusBadRequest,
				message:     "failed to parse JSON: invalid character 'b' after object key",
			},
			{
				description: "with a request body exceeding the maximum size",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"isbn": %q, "title": %q, "author": %q}`,
					util.RandomString(10_000_000),
					util.RandomString(10_000_000),
					util.RandomString(10_000_000),
				)),
				statusCode: http.StatusRequestEntityTooLarge,
				message:    "http: request body too large",
			},
			{
				description: "with a missing book ISBN",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"title": %q, "author": %q}`,
					util.RandomString(10),
					util.RandomString(10),
				)),
				statusCode: http.StatusBadRequest,
				message:    "ISBN, title, and author must not be empty.",
			},
			{
				description: "with a missing book title",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"isbn": %q, "author": %q}`,
					util.RandomString(10),
					util.RandomString(10),
				)),
				statusCode: http.StatusBadRequest,
				message:    "ISBN, title, and author must not be empty.",
			},
			{
				description: "with a missing book author",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"isbn": %q, "title": %q}`,
					util.RandomString(10),
					util.RandomString(10),
				)),
				statusCode: http.StatusBadRequest,
				message:    "ISBN, title, and author must not be empty.",
			},
		}

		for _, test := range tests {
			t.Run(test.description, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, routeBooks, test.reqBody)
				req.Header.Set(headerKeyContentType, headerValueApplicationJSON)
				w := httptest.NewRecorder()

				var beforeBooks, afterBooks []db.Book
				withStartedTestServer(t, func(s *Server) {
					var err error

					ctx := context.Background()
					beforeBooks, err = s.libraryDB.Books(ctx)
					require.Nil(t, err)

					s.handleCreateBook(w, req)

					afterBooks, err = s.libraryDB.Books(ctx)
					require.Nil(t, err)
				})

				t.Run("responds with the expected status code", func(t *testing.T) {
					assert.Equal(t, test.statusCode, w.Code)
				})

				t.Run("responds with valid JSON", func(t *testing.T) {
					var resp Response
					assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

					t.Run("with the expected message", func(t *testing.T) {
						assert.Equal(t, test.message, resp.Message)
					})
				})

				t.Run("does not insert a new book into the database", func(t *testing.T) {
					assert.Equal(t, beforeBooks, afterBooks)
				})
			})
		}
	})

	t.Run("when the database insert fails", func(t *testing.T) {
		t.Skip()
	})
}

func TestServer_handleCheckInBook(t *testing.T) {
	tests := []struct {
		description        string
		bookID             string
		expectedMessage    string
		expectedStatusCode int
	}{
		{
			description:        "with an existing book ID",
			bookID:             "1",
			expectedMessage:    "Book checked in.",
			expectedStatusCode: http.StatusOK,
		},
		{
			description:        "with a book ID that does not exist",
			bookID:             "1000000",
			expectedMessage:    "Book with ID 1000000 not found.",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			description:        "with an invalid book ID value",
			bookID:             "foo",
			expectedMessage:    "Invalid book ID. Must be an integer.",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPatch,
				fmt.Sprintf("%s/check-in/%s", routeBooks, test.bookID),
				nil,
			)
			req.SetPathValue("id", test.bookID)
			w := httptest.NewRecorder()

			withStartedTestServer(t, func(s *Server) {
				body, err := json.Marshal(Book{
					Isbn:   util.RandomString(10),
					Title:  util.RandomString(10),
					Author: util.RandomString(10),
				})
				require.Nil(t, err)

				s.handleCreateBook(
					httptest.NewRecorder(),
					httptest.NewRequest(http.MethodPost, routeBooks, bytes.NewBuffer(body)),
				)

				s.handleCheckOutBook(
					httptest.NewRecorder(),
					httptest.NewRequest(
						http.MethodPatch,
						fmt.Sprintf("%s/check-out/1", routeBooks),
						nil,
					),
				)

				s.handleCheckInBook(w, req)
			})

			t.Run("responds with the expected status code", func(t *testing.T) {
				assert.Equal(t, test.expectedStatusCode, w.Code)
			})

			t.Run("responds with valid JSON", func(t *testing.T) {
				var resp Response
				assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

				t.Run("with the expected content", func(t *testing.T) {
					assert.Equal(t, test.expectedMessage, resp.Message)

					if resp.Data != nil {
						var respBook Book
						require.Nil(t, json.Unmarshal(resp.Data, &respBook))

						assert.Zero(t, respBook.CheckedOutAt.Time)
						assert.False(t, respBook.CheckedOutAt.Valid)
					}
				})
			})
		})
	}
}

func TestServer_handleCheckOutBook(t *testing.T) {
	tests := []struct {
		description        string
		bookID             string
		expectedMessage    string
		expectedStatusCode int
	}{
		{
			description:        "with an existing book ID",
			bookID:             "1",
			expectedMessage:    "Book checked out.",
			expectedStatusCode: http.StatusOK,
		},
		{
			description:        "with a book ID that does not exist",
			bookID:             "1000000",
			expectedMessage:    "Book with ID 1000000 not found.",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			description:        "with an invalid book ID value",
			bookID:             "foo",
			expectedMessage:    "Invalid book ID. Must be an integer.",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPatch,
				fmt.Sprintf("%s/check-out/%s", routeBooks, test.bookID),
				nil,
			)
			req.SetPathValue("id", test.bookID)
			w := httptest.NewRecorder()

			withStartedTestServer(t, func(s *Server) {
				body, err := json.Marshal(Book{
					Isbn:   util.RandomString(10),
					Title:  util.RandomString(10),
					Author: util.RandomString(10),
				})
				require.Nil(t, err)

				s.handleCreateBook(
					httptest.NewRecorder(),
					httptest.NewRequest(http.MethodPost, routeBooks, bytes.NewBuffer(body)),
				)

				s.handleCheckOutBook(w, req)
			})

			t.Run("responds with the expected status code", func(t *testing.T) {
				assert.Equal(t, test.expectedStatusCode, w.Code)
			})

			t.Run("responds with valid JSON", func(t *testing.T) {
				var resp Response
				assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

				t.Run("with the expected content", func(t *testing.T) {
					assert.Equal(t, test.expectedMessage, resp.Message)

					if resp.Data != nil {
						var respBook Book
						require.Nil(t, json.Unmarshal(resp.Data, &respBook))

						assert.NotZero(t, respBook.CheckedOutAt.Time)
						assert.True(t, respBook.CheckedOutAt.Valid)
					}
				})
			})
		})
	}
}

func TestServer_handleDeleteBook(t *testing.T) {
	tests := []struct {
		description        string
		bookID             string
		expectedMessage    string
		expectedStatusCode int
	}{
		{
			description:        "with an existing book ID",
			bookID:             "1",
			expectedMessage:    "Deleted book with ID 1.",
			expectedStatusCode: http.StatusOK,
		},
		{
			description:        "with a book ID that does not exist",
			bookID:             "1000000",
			expectedMessage:    "Book with ID 1000000 not found.",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			description:        "with an invalid book ID value",
			bookID:             "foo",
			expectedMessage:    "Invalid book ID. Must be an integer.",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodDelete,
				fmt.Sprintf("%s/%s", routeBooks, test.bookID),
				nil,
			)
			req.SetPathValue("id", test.bookID)
			w := httptest.NewRecorder()

			withStartedTestServer(t, func(s *Server) {
				body, err := json.Marshal(Book{
					Isbn:   util.RandomString(10),
					Title:  util.RandomString(10),
					Author: util.RandomString(10),
				})
				require.Nil(t, err)

				s.handleCreateBook(
					httptest.NewRecorder(),
					httptest.NewRequest(http.MethodPost, routeBooks, bytes.NewBuffer(body)),
				)

				s.handleDeleteBook(w, req)
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

func TestServer_handleGetBook(t *testing.T) {
	tests := []struct {
		description        string
		bookID             string
		expectedData       bool
		expectedMessage    string
		expectedStatusCode int
	}{
		{
			description:        "with an existing book ID",
			bookID:             "1",
			expectedData:       true,
			expectedMessage:    "Book found.",
			expectedStatusCode: http.StatusOK,
		},
		{
			description:        "with an book ID that does not exist",
			bookID:             "1000000",
			expectedData:       false,
			expectedMessage:    "Book with ID 1000000 not found.",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			description:        "with an invalid book ID value",
			bookID:             "foo",
			expectedData:       false,
			expectedMessage:    "Invalid book ID. Must be an integer.",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("%s/%s", routeBooks, test.bookID),
				nil,
			)
			req.SetPathValue("id", test.bookID)
			w := httptest.NewRecorder()

			withStartedTestServer(t, func(s *Server) {
				body, err := json.Marshal(Book{
					Isbn:   util.RandomString(10),
					Title:  util.RandomString(10),
					Author: util.RandomString(10),
				})
				require.Nil(t, err)

				s.handleCreateBook(
					httptest.NewRecorder(),
					httptest.NewRequest(http.MethodPost, routeBooks, bytes.NewBuffer(body)),
				)

				s.handleGetBook(w, req)
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
						var respBook Book
						require.Nil(t, json.Unmarshal(resp.Data, &respBook))

						assert.NotEmpty(t, respBook)
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

func TestServer_handleListBooks(t *testing.T) {
	t.Run("with a valid request", func(t *testing.T) {
		tests := []struct {
			description     string
			expectedResults int
			testBooks       []Book
		}{
			{
				description:     "with multiple books",
				expectedResults: 2,
				testBooks: []Book{
					{
						Isbn:   util.RandomString(10),
						Title:  util.RandomString(10),
						Author: util.RandomString(10),
					},
					{
						Isbn:   util.RandomString(10),
						Title:  util.RandomString(10),
						Author: util.RandomString(10),
					},
				},
			},
			{
				description:     "with an archived book",
				expectedResults: 1,
				testBooks: []Book{
					{
						Isbn:   util.RandomString(10),
						Title:  util.RandomString(10),
						Author: util.RandomString(10),
					},
					{
						Isbn:   util.RandomString(10),
						Title:  util.RandomString(10),
						Author: util.RandomString(10),
						ArchivedAt: sql.NullTime{
							Time:  time.Now().UTC(),
							Valid: true,
						},
					},
				},
			},
			{
				description:     "with no books found",
				expectedResults: 0,
				testBooks:       []Book{},
			},
		}

		for _, test := range tests {
			req := httptest.NewRequest(http.MethodGet, routeBooks, nil)
			w := httptest.NewRecorder()

			withStartedTestServer(t, func(s *Server) {
				for _, book := range test.testBooks {
					body, err := json.Marshal(book)
					require.Nil(t, err)

					s.handleCreateBook(
						httptest.NewRecorder(),
						httptest.NewRequest(http.MethodPost, routeBooks, bytes.NewBuffer(body)),
					)
				}

				s.handleListBooks(w, req)
			})

			t.Run(test.description, func(t *testing.T) {
				t.Run("responds with the expected status code", func(t *testing.T) {
					assert.Equal(t, http.StatusOK, w.Code)
				})

				t.Run("responds with valid JSON", func(t *testing.T) {
					var resp Response
					assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

					t.Run("with the expected content", func(t *testing.T) {
						assert.Equal(t, "Listed books.", resp.Message)

						var respBooks []Book
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

func TestServer_handleUpdateBook(t *testing.T) {
	t.Run("with a valid request", func(t *testing.T) {
		testBody := UpdateBookRequest{
			ID:     1,
			Isbn:   util.RandomString(10),
			Title:  util.RandomString(10),
			Author: util.RandomString(10),
		}

		body, err := json.Marshal(testBody)
		require.Nil(t, err)

		req := httptest.NewRequest(http.MethodPut, routeBooks, bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		withStartedTestServer(t, func(s *Server) {
			body, err := json.Marshal(Book{
				Isbn:   util.RandomString(10),
				Title:  util.RandomString(10),
				Author: util.RandomString(10),
			})
			require.Nil(t, err)

			s.handleCreateBook(
				httptest.NewRecorder(),
				httptest.NewRequest(http.MethodPost, routeBooks, bytes.NewBuffer(body)),
			)

			s.handleUpdateBook(w, req)
		})

		t.Run("responds with the expected status code", func(t *testing.T) {
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("responds with valid JSON", func(t *testing.T) {
			var resp Response
			assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

			t.Run("with the expected content", func(t *testing.T) {
				assert.Equal(t, "Book updated.", resp.Message)

				var respBook Book
				require.Nil(t, json.Unmarshal(resp.Data, &respBook))

				assert.NotEmpty(t, respBook)
				assert.Equal(t, int64(1), respBook.ID)
				assert.Equal(t, testBody.Isbn, respBook.Isbn)
				assert.Equal(t, testBody.Title, respBook.Title)
				assert.Equal(t, testBody.Author, respBook.Author)
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
				description: "with a book ID that does not exist",
				reqBody: bytes.NewBufferString(fmt.Sprintf(
					`{"id": 1000000, "author": %q}`,
					util.RandomString(10),
				)),
				expectedMessage:    "Book with ID 1000000 not found.",
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
					`{"id": 1, "isbn": %q, "title": %q, "author": %q}`,
					util.RandomString(10_000_000),
					util.RandomString(10_000_000),
					util.RandomString(10_000_000),
				)),
				expectedStatusCode: http.StatusRequestEntityTooLarge,
				expectedMessage:    "http: request body too large",
			},
		}

		for _, test := range tests {
			t.Run(test.description, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPut, routeBooks, test.reqBody)
				w := httptest.NewRecorder()

				withStartedTestServer(t, func(s *Server) {
					body, err := json.Marshal(Book{
						Isbn:   util.RandomString(10),
						Title:  util.RandomString(10),
						Author: util.RandomString(10),
					})
					require.Nil(t, err)

					s.handleCreateBook(
						httptest.NewRecorder(),
						httptest.NewRequest(http.MethodPost, routeBooks, bytes.NewBuffer(body)),
					)

					s.handleUpdateBook(w, req)
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

		t.Run("with an existing book ISBN", func(t *testing.T) {
			testISBN := util.RandomString(10)
			testBody := UpdateBookRequest{
				ID:     1,
				Isbn:   testISBN,
				Title:  util.RandomString(10),
				Author: util.RandomString(10),
			}

			body, err := json.Marshal(testBody)
			require.Nil(t, err)

			req := httptest.NewRequest(http.MethodPut, routeBooks, bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			withStartedTestServer(t, func(s *Server) {
				firstBody, err := json.Marshal(Book{
					Isbn:   util.RandomString(10),
					Title:  util.RandomString(10),
					Author: util.RandomString(10),
				})
				require.Nil(t, err)

				s.handleCreateBook(
					httptest.NewRecorder(),
					httptest.NewRequest(http.MethodPost, routeBooks, bytes.NewBuffer(firstBody)),
				)

				secondBody, err := json.Marshal(Book{
					Isbn:   testISBN,
					Title:  util.RandomString(10),
					Author: util.RandomString(10),
				})
				require.Nil(t, err)

				s.handleCreateBook(
					httptest.NewRecorder(),
					httptest.NewRequest(http.MethodPost, routeBooks, bytes.NewBuffer(secondBody)),
				)

				s.handleUpdateBook(w, req)
			})

			t.Run("responds with the expected status code", func(t *testing.T) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			})

			t.Run("responds with valid JSON", func(t *testing.T) {
				var resp Response
				assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &resp))

				t.Run("with the expected content", func(t *testing.T) {
					assert.Equal(t, fmt.Sprintf("A book with ISBN %q already exists.", testISBN), resp.Message)
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

func withStartedTestServer(t *testing.T, test func(*Server)) {
	t.Helper()

	withTestServer(t, func(s *Server) {
		time.AfterFunc(500*time.Millisecond, func() {
			test(s)
			require.Nil(t, s.Stop(make(chan struct{})))
		})

		require.Nil(t, s.Start())
	})
}
