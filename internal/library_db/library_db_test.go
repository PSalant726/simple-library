package library_db

import (
	"database/sql"
	"os"
	"testing"

	db "github.com/PSalant726/simple-library/db/sqlc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pathMigrations = "../../db/migrations"

func TestNew(t *testing.T) {
	t.Run("returns the expected LibraryDB", func(t *testing.T) {
		tmpDB, err := os.CreateTemp(os.TempDir(), "test_db_*.sqlite")
		require.Nil(t, err)
		defer os.Remove(tmpDB.Name())

		testDB, err := New(tmpDB.Name(), pathMigrations)
		assert.Nil(t, err)
		assert.IsType(t, &LibraryDB{}, testDB)
		assert.Equal(t, cfgBusyTimeout, testDB.busyTimeout)
		assert.Equal(t, cfgCacheSize, testDB.cacheSize)
		assert.Equal(t, cfgJournalMode, testDB.journalMode)
		assert.Equal(t, databaseName, testDB.name)
		assert.Equal(t, tmpDB.Name(), testDB.pathDBFile)
		assert.Equal(t, pathMigrations, testDB.pathMigrationsDir)
		assert.Equal(t, cfgSynchronous, testDB.synchronous)
	})

	t.Run("returns an error", func(t *testing.T) {
		t.Run("when the database file cannot be opened", func(t *testing.T) {
			testDB, err := New("/nonexistent", pathMigrations)
			assert.Nil(t, testDB)
			assert.Error(t, err)
		})

		t.Run("when the migrations directory does not exist", func(t *testing.T) {
			tmpDB, err := os.CreateTemp(os.TempDir(), "test_db_*.sqlite")
			require.Nil(t, err)
			defer os.Remove(tmpDB.Name())

			testDB, err := New(tmpDB.Name(), "/nonexistent")
			assert.Nil(t, testDB)
			assert.Error(t, err)
		})

		t.Run("when the migrations directory is not a directory", func(t *testing.T) {
			tmpDB, err := os.CreateTemp(os.TempDir(), "test_db_*.sqlite")
			require.Nil(t, err)
			defer os.Remove(tmpDB.Name())

			testDB, err := New(tmpDB.Name(), tmpDB.Name())
			assert.Nil(t, testDB)
			assert.Error(t, err)
		})
	})
}

func TestLibraryDB_Start(t *testing.T) {
	t.Run("returns nil", func(t *testing.T) {
		withTempDB(t, func(d *LibraryDB) {
			assert.Nil(t, d.Start())
		})
	})

	t.Run("sets the Queries field", func(t *testing.T) {
		withTempDB(t, func(d *LibraryDB) {
			assert.Nil(t, d.Start())
			assert.IsType(t, &db.Queries{}, d.Queries)
		})
	})

	t.Run("sets the db field", func(t *testing.T) {
		withTempDB(t, func(d *LibraryDB) {
			assert.Nil(t, d.Start())
			assert.IsType(t, &sql.DB{}, d.db)
		})
	})

	t.Run("returns an error", func(t *testing.T) {
		t.Run("when a database connection cannot be established", func(t *testing.T) {
			withTempDB(t, func(d *LibraryDB) {
				d.pathDBFile = "/nonexistent"
				assert.Error(t, d.Start())
			})
		})

		t.Run("when the database cannot be migrated", func(t *testing.T) {
			withTempDB(t, func(d *LibraryDB) {
				d.pathMigrationsDir = "/nonexistent"
				assert.Error(t, d.Start())
			})
		})
	})
}

func TestLibraryDB_Stop(t *testing.T) {
	t.Run("returns nil", func(t *testing.T) {
		withTempDB(t, func(d *LibraryDB) {
			require.Nil(t, d.Start())
			assert.Nil(t, d.Stop())
		})
	})

	t.Run("returns an error", func(t *testing.T) {
		t.Run("when the database connection cannot be closed", func(t *testing.T) {
			t.Skip("Getting modernc.org/sqlite to return an error on Close() is hard!")

			withTempDB(t, func(d *LibraryDB) {
				require.Nil(t, d.Start())
				require.Nil(t, d.Stop())

				assert.Error(t, d.Stop())
			})
		})
	})
}

func withTempDB(t *testing.T, test func(*LibraryDB)) {
	t.Helper()

	tmpDB, err := os.CreateTemp(os.TempDir(), "test_db_*.sqlite")
	require.Nil(t, err)
	defer os.Remove(tmpDB.Name())

	testDB, err := New(tmpDB.Name(), pathMigrations)
	require.Nil(t, err)
	defer func() {
		if testDB.db == nil {
			return
		}

		if err := testDB.db.Ping(); err != nil {
			testDB.db.Close()
		}
	}()

	test(testDB)
}
