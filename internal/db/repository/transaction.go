package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ky0ryu/video-upload-service/internal/sqlc/model"
)

type Transaction struct {
	db *pgxpool.Pool
}

func NewTransaction(db *pgxpool.Pool) *Transaction {
	return &Transaction{
		db: db,
	}
}

func (t *Transaction) ExecTx(ctx context.Context, fn func(*sqlc.Queries) error) error {
	tx, err := t.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		// use a new context.Background() Rollback can communicate with the DB
		// if incase ctx have already timeout-out or have been cancelled
		if rbErr := tx.Rollback(context.Background()); rbErr != nil && rbErr != pgx.ErrTxClosed {
			log.Printf("unexpected rollback error: %v (original transaction error: %v)", rbErr, err)
		}
	}()

	// execute the DB operations thru fn()
	q := sqlc.New(tx)
	if err = fn(q); err != nil {
		return fmt.Errorf("tx fn failed: %w", err)
	}

	// reflect changes to DB
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}
