package db

import (
	"context"
	"database/sql"
)

func WithTx(db *sql.DB, ctx context.Context, fn func(*sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	err = fn(tx)
	if err != nil {
		return err
	}
	return tx.Commit()
}
