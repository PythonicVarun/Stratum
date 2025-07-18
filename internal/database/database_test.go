package database

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestIsValidIdentifier(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid lowercase", "table_name", true},
		{"Valid uppercase", "COLUMN_NAME", true},
		{"Valid mixed case", "TableName", true},
		{"Valid with numbers", "table1", true},
		{"Valid with underscore prefix", "_table", true},
		{"Empty string", "", false},
		{"Contains hyphen", "table-name", false},
		{"Contains space", "table name", false},
		{"Contains semicolon", "table;", false},
		{"SQL injection attempt", "'; DROP TABLE users; --", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isValidIdentifier(tc.input))
		})
	}
}

func TestGenericDB_Fetch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	t.Run("Invalid Identifier", func(t *testing.T) {
		gdb := &GenericDB{db: db}
		_, err := gdb.Fetch("invalid-table", "id", "data", "1")
		assert.Error(t, err)
		assert.Equal(t, "invalid table or column name", err.Error())
	})

	t.Run("Successful Fetch MySQL", func(t *testing.T) {
		gdb := &GenericDB{db: db, driverName: "mysql"}
		rows := sqlmock.NewRows([]string{"data"}).AddRow([]byte("test_data"))
		mock.ExpectQuery("SELECT data FROM users WHERE id = ?").WithArgs("1").WillReturnRows(rows)

		data, err := gdb.Fetch("users", "id", "data", "1")
		assert.NoError(t, err)
		assert.Equal(t, []byte("test_data"), data)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Successful Fetch Postgres", func(t *testing.T) {
		gdb := &GenericDB{db: db, driverName: "postgres"}
		rows := sqlmock.NewRows([]string{"data"}).AddRow([]byte("test_data_pg"))
		mock.ExpectQuery("SELECT data FROM users WHERE id = \\$1").WithArgs("2").WillReturnRows(rows)

		data, err := gdb.Fetch("users", "id", "data", "2")
		assert.NoError(t, err)
		assert.Equal(t, []byte("test_data_pg"), data)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("No Rows Found", func(t *testing.T) {
		gdb := &GenericDB{db: db, driverName: "mysql"}
		mock.ExpectQuery("SELECT data FROM users WHERE id = ?").WithArgs("3").WillReturnError(sql.ErrNoRows)

		data, err := gdb.Fetch("users", "id", "data", "3")
		assert.NoError(t, err)
		assert.Nil(t, data)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Query Error", func(t *testing.T) {
		gdb := &GenericDB{db: db, driverName: "mysql"}
		mock.ExpectQuery("SELECT data FROM users WHERE id = ?").WithArgs("4").WillReturnError(errors.New("db error"))

		_, err := gdb.Fetch("users", "id", "data", "4")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database query failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// Note: Testing NewDBLoader and ConnectionManager is complex due to the direct
// use of sql.Open and the lack of dependency injection for the DBLoader constructor.
// A refactor would be needed to make these components more testable.
// Given the constraint not to change project logic, these tests are omitted.
