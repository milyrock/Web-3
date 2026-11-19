package login

import (
	"context"
	"net/http"

	"developer.dev/web-3/internal/handlers/response"
	"developer.dev/web-3/internal/handlers/session"
	"developer.dev/web-3/internal/model"
)

type Usecase interface {
	Login(ctx context.Context, email, password string) (model.Session, error)
}

type Request struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func New(service Usecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input Request
		if err := response.Decode(w, r, &input); err != nil {
			response.Error(w, err)
			return
		}
		auth, err := service.Login(r.Context(), input.Email, input.Password)
		if err != nil {
			response.Error(w, err)
			return
		}
		session.Set(w, r, auth)
		response.Send(w, http.StatusOK, response.NewUser(auth.User), nil)
	}
}
