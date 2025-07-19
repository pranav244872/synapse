package db

import (
	"context"
	"os"
	"log"
	"testing"
	"github.com/jackc/pgx/v5"
)

const (
	dbSource = "postgres://postgres:secret@postgresDB:5432/synapse?sslmode=disable"
)

var testQueries *Queries;

/*
By convention the TestMain function is the main entry point
of all unit tests inside 1 specific golang package
*/

func TestMain(m *testing.M) {
	conn, err := pgx.Connect(context.Background(), dbSource)	 
	if err != nil {
		log.Fatal("cannot connect to db:", err)
		os.Exit(1)
	}

	testQueries = New(conn)

	/*
	m.Run() will run all the unit tests and return exit codes which tell
	us whether the tests pass or fail
	*/
	os.Exit(m.Run())
}
