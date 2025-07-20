package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"fmt"
)

// Store provides all functions to execute db queries and transactions.
// It now holds a pgxpool.Pool, which satisfies the DBTX interface.
type Store struct {
    *Queries
    dbpool *pgxpool.Pool
}

// NewStore creates a new Store.
func NewStore(dbpool *pgxpool.Pool) *Store {
    return &Store{
        dbpool:  dbpool,
        Queries: New(dbpool),
    }
}

// execTx executes a function within a database transaction.
// This is a common helper function to add to the Store struct.
func (s *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
    tx, err := s.dbpool.Begin(ctx)
    if err != nil {
        return err
    }

    q := New(tx)
    err = fn(q)
    if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
        return err
    }

    return tx.Commit(ctx)
}
