package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ky0ryu/video-upload-service/internal/db/model"
)

type Transaction struct {
	db *pgxpool.Pool
}

func NewTransaction(db *pgxpool.Pool) *Transaction {
	return &Transaction{
		db: db,
	}
}

func (t *Transaction) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := t.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr != pgx.ErrTxClosed {
			log.Printf("unexpected rollback error: %v (original transaction error: %v)", rbErr, err)
		}
	}()

	q := db.New(tx)
	if err = fn(q); err != nil {
		return fmt.Errorf("tx fn failed: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}
