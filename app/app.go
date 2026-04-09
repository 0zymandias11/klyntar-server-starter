package app

import (
	"database/sql"
	"log/slog"
	"net/http"

	"example.com/klyntar-server/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Application struct {
	Config             config.Config
	DbConnector        *sql.DB
	routeRegistar      []RouteRegister
	soloRouteRegistrar []RouteRegister
}

type RouteRegister func(r chi.Router, app *Application)

func (app *Application) RegisterRoutes(registrars ...RouteRegister) {
	app.routeRegistar = append(app.routeRegistar, registrars...)
}

func (app *Application) RegisterSoloRoutes(registars ...RouteRegister) {
	app.soloRouteRegistrar = append(app.soloRouteRegistrar, registars...)
}

func (app *Application) Mount() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", app.healthCheckHandler)

	r.Route("/api/v1", func(r chi.Router) {
		for _, registrar := range app.routeRegistar {
			registrar(r, app)
		}
	})

	slog.Info("Registering Solo routes", "count", len(app.soloRouteRegistrar))
	for _, registrar := range app.soloRouteRegistrar {
		registrar(r, app)
	}

	chi.Walk(r, func(method string, route string, handler http.Handler,
		middlewares ...func(http.Handler) http.Handler) error {
		slog.Info("Registered route", "method", method, "route", route)
		return nil
	})

	return r
}

func (app *Application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (app *Application) Run(mux *chi.Mux) error {
	srv := &http.Server{
		Addr:    app.Config.Addr,
		Handler: mux,
	}
	slog.Info("server started at ", app.Config.Addr, "Default")
	return srv.ListenAndServe()
}
