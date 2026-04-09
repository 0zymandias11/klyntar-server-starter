package routes

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"example.com/klyntar-server/api/controllers"
	"example.com/klyntar-server/app"
	"example.com/klyntar-server/models"
	"example.com/klyntar-server/utils"
	"github.com/go-chi/chi/v5"
)

type UserPayload struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Key_fp   string `json:"key_fp"`
	Dvc_mac  string `json:"dvc_mac"`
}

func RegisterUserRoutes(r chi.Router, application *app.Application) {
	r.Post("/user", getUserHandler(application))
	r.Post("/createUser", createUserHandler(application))
}

func getUserHandler(application *app.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request := new(UserPayload)
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			utils.WriteJson(w, http.StatusBadRequest, "Invalid login request")
			return
		}

		user := new(models.User)
		user.Key_fp = request.Key_fp

		result, err := controllers.GetOneUser(*user, r, *application)
		if err != nil {
			slog.Error("Error Getting User", "err", err)
			utils.WriteJson(w, http.StatusNotFound, "User not found")
			return
		}
		slog.Info("User fetch Success")
		utils.WriteJson(w, http.StatusOK, result)
	}
}

func createUserHandler(application *app.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request := new(UserPayload)
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			utils.WriteJson(w, http.StatusBadRequest, "Invalid User Creation Request")
			return
		}

		user := new(models.User)
		user.Email = request.Email
		user.Username = request.Username
		user.Key_fp = request.Key_fp
		user.Dvc_mac = request.Dvc_mac

		err := controllers.CreateOneUser(*user, r, *application)
		if err != nil {
			slog.Error("Error Creating User", "err", err)
			utils.WriteJson(w, http.StatusBadRequest, "Cannot Create User !!")
			return
		}
		slog.Info("User Creation success")
		utils.WriteJson(w, http.StatusOK, "")
	}
}
