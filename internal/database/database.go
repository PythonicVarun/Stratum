package database

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// DBLoader defines the interface for fetching data from a database.
type DBLoader interface {
	Fetch(table, idColumn, serveColumn, idValue string) ([]byte, error)
	Close()
}

// GenericDB is a concrete implementation of DBLoader for SQL databases.
type GenericDB struct {
	db         *sql.DB
	driverName string
}

var validIdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// NewDBLoader creates and returns a new DBLoader instance for the given DSN.
// It currently supports "postgres" and "mysql".
func NewDBLoader(dsn string) (DBLoader, error) {
	var driverName string
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		driverName = "postgres"
	} else if strings.Contains(dsn, "@tcp(") {
		driverName = "mysql"
	} else {
		return nil, fmt.Errorf("unsupported database dialect for DSN")
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Printf("Successfully connected to %s database.", driverName)
	return &GenericDB{db: db, driverName: driverName}, nil
}

func (g *GenericDB) Fetch(table, idColumn, serveColumn, idValue string) ([]byte, error) {
	if !isValidIdentifier(table) || !isValidIdentifier(idColumn) || !isValidIdentifier(serveColumn) {
		return nil, fmt.Errorf("invalid table or column name")
	}

	// Securely quote identifiers
	quotedTable := g.QuoteIdentifier(table)
	quotedIDColumn := g.QuoteIdentifier(idColumn)
	quotedServeColumn := g.QuoteIdentifier(serveColumn)

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", quotedServeColumn, quotedTable, quotedIDColumn)
	if g.driverName == "postgres" {
		query = strings.Replace(query, "?", "$1", 1)
	}

	var result []byte
	err := g.db.QueryRow(query, idValue).Scan(&result)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	return result, nil
}

// Close closes the database connection.
func (g *GenericDB) Close() {
	if g.db != nil {
		g.db.Close()
	}
}

// QuoteIdentifier wraps an identifier in the correct quotes for the database driver.
func (g *GenericDB) QuoteIdentifier(name string) string {
	switch g.driverName {
	case "postgres":
		return `"` + name + `"`
	case "mysql":
		return "`" + name + "`"
	default:
		// Fallback for unknown drivers, though we might want to be stricter
		return name
	}
}

// Checks if a string is a valid SQL identifier (table or column name).
func isValidIdentifier(name string) bool {
	return validIdentifierRegex.MatchString(name)
}

// ConnectionManager handles multiple database connections.
type ConnectionManager struct {
	connections map[string]DBLoader
	mu          sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]DBLoader),
	}
}

// Get returns an existing database connection or creates a new one.
func (cm *ConnectionManager) Get(dsn string) (DBLoader, error) {
	cm.mu.RLock()
	conn, ok := cm.connections[dsn]
	cm.mu.RUnlock()

	if ok {
		return conn, nil
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	conn, ok = cm.connections[dsn]
	if ok {
		return conn, nil
	}

	newConn, err := NewDBLoader(dsn)
	if err != nil {
		return nil, err
	}

	cm.connections[dsn] = newConn
	return newConn, nil
}

// CloseAll closes all managed database connections.
func (cm *ConnectionManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for _, conn := range cm.connections {
		conn.Close()
	}
}
