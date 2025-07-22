// db/main_test.go
package db

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	dbSource = "postgres://postgres:secret@postgresDB:5432/synapse?sslmode=disable"
)

// We keep testQueries for direct, simple queries in our tests.
var testQueries *Queries
// testPool is the new, crucial addition. It's a connection pool.
var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	var err error

	// pgx.Connect (the old way) creates a SINGLE database connection. This is bad for
	// a web server and cannot handle concurrent requests.
	// pgxpool.New (the new way) creates a POOL of database connections. When a request
	// needs a connection, the pool lends one out and takes it back when done.
	// This is essential for handling concurrent operations, both in your real app and
	// in your tests.
	testPool, err = pgxpool.New(context.Background(), dbSource)
	if err != nil {
		log.Fatalf("cannot create db pool: %v", err)
		os.Exit(1)
	}

	// We can still create a Queries object from the pool for convenience.
	testQueries = New(testPool)

	os.Exit(m.Run())
}
