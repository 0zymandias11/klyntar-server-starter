package utils

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func WriteJson(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to write json response", "err", err)
	}
}
