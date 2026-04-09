package controllers

import (
	"database/sql"
	"net/http"

	"example.com/klyntar-server/app"
	"example.com/klyntar-server/models"
	"example.com/klyntar-server/pkg/db"
)

func GetOneUser(user models.User, r *http.Request, application app.Application) (*models.User, error) {
	var result models.User
	txErr := db.WithTx(application.DbConnector, r.Context(), func(tx *sql.Tx) error {
		err := tx.QueryRowContext(r.Context(),
			"SELECT id, username, email, device_mac FROM users WHERE key_fingerprint = $1",
			user.Key_fp,
		).Scan(&result.ID, &result.Username, &result.Email, &result.Dvc_mac)
		return err
	})
	if txErr != nil {
		return nil, txErr
	}
	return &result, nil
}

func CreateOneUser(user models.User, r *http.Request, application app.Application) error {
	txErr := db.WithTx(application.DbConnector, r.Context(), func(tx *sql.Tx) error {
		_, err := tx.ExecContext(r.Context(),
			"INSERT INTO users (username, email, key_fingerprint, device_mac) VALUES ($1, $2, $3, $4)",
			user.Username, user.Email, user.Key_fp, user.Dvc_mac,
		)
		return err
	})
	return txErr
}
