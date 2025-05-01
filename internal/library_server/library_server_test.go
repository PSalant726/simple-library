package library_server

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/PSalant726/simple-library/internal/library_db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("returns the expected Server", func(t *testing.T) {
		tmpDB, err := os.CreateTemp(os.TempDir(), "test_db_*.sqlite")
		require.Nil(t, err)
		defer os.Remove(tmpDB.Name())

		testServer, err := New("localhost", 8080, tmpDB.Name(), "../../db/migrations", false)
		assert.Nil(t, err)
		assert.IsType(t, &Server{}, testServer)
		assert.Equal(t, "localhost:8080", testServer.Addr)
		assert.Equal(t, defaultServerTimeout, testServer.ReadTimeout)
		assert.Equal(t, defaultServerTimeout, testServer.ReadHeaderTimeout)
		assert.Equal(t, defaultServerTimeout, testServer.WriteTimeout)
		assert.IsType(t, &library_db.LibraryDB{}, testServer.libraryDB)
		assert.NotNil(t, testServer.Handler)
	})

	t.Run("returns an error", func(t *testing.T) {
		t.Run("when the library database cannot be created", func(t *testing.T) {
			testServer, err := New("localhost", 8080, "/nonexistent", "/nonexistent", false)
			assert.Nil(t, testServer)
			assert.Error(t, err)
		})
	})
}

func TestServer_Start(t *testing.T) {
	t.Run("returns nil", func(t *testing.T) {
		withTestServer(t, func(s *Server) {
			time.AfterFunc(500*time.Millisecond, func() {
				require.Nil(t, s.Stop(make(chan struct{})))
			})

			assert.EventuallyWithT(t, func(c *assert.CollectT) {
				assert.Nil(c, s.Start())
			}, time.Second, time.Millisecond)
		})
	})

	t.Run("returns an error", func(t *testing.T) {
		t.Run("when the library database cannot be started", func(t *testing.T) {
			withTestServer(t, func(s *Server) {
				s.libraryDB = &library_db.LibraryDB{}
				assert.Error(t, s.Start())
			})
		})

		t.Run("when the server fails to listen", func(t *testing.T) {
			t.Skip("Getting ListenAndServe() to return an error other than ErrServerClosed is hard!")

			withTestServer(t, func(s *Server) {
				assert.Error(t, s.Start())
			})
		})
	})
}

func TestServer_Stop(t *testing.T) {
	t.Run("returns nil", func(t *testing.T) {
		withTestServer(t, func(s *Server) {
			time.AfterFunc(500*time.Millisecond, func() {
				assert.Nil(t, s.Stop(make(chan struct{})))
			})

			require.Nil(t, s.Start())
		})
	})

	t.Run("returns an error", func(t *testing.T) {
		t.Run("when the library server cannot be shut down", func(t *testing.T) {
			t.Skip("Getting Shutdown() to return an error is hard!")

			withTestServer(t, func(s *Server) {
				time.AfterFunc(500*time.Millisecond, func() {
					require.Nil(t, s.Shutdown(context.Background()))
					assert.Error(t, s.Stop(make(chan struct{})))
				})

				require.Nil(t, s.Start())
			})
		})

		t.Run("when the library database connection cannot be closed", func(t *testing.T) {
			t.Skip("Getting modernc.org/sqlite to return an error on Close() is hard!")

			withTestServer(t, func(s *Server) {
				time.AfterFunc(500*time.Millisecond, func() {
					assert.Error(t, s.Stop(make(chan struct{})))
				})

				require.Nil(t, s.Start())
				require.Nil(t, s.libraryDB.Stop())
			})
		})
	})
}

func withTestServer(t *testing.T, test func(*Server)) {
	t.Helper()

	tmpDB, err := os.CreateTemp(os.TempDir(), "test_db_*.sqlite")
	require.Nil(t, err)
	defer os.Remove(tmpDB.Name())

	testServer, err := New("localhost", 8080, tmpDB.Name(), "../../db/migrations", false)
	require.Nil(t, err)

	test(testServer)
}
