package routes

import (
	"net/http"

	"example.com/klyntar-server/app"
	"github.com/go-chi/chi/v5"
)

func RegisterHealthRoutes(r chi.Router, application *app.Application) {
	r.Get("/health", healthCheckHandler(application))
}

func healthCheckHandler(app *app.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}
