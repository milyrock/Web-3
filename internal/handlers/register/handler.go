package register

import (
	"context"
	"net/http"

	"developer.dev/web-3/internal/handlers/response"
	"developer.dev/web-3/internal/handlers/session"
	"developer.dev/web-3/internal/model"
)

type Usecase interface {
	Register(ctx context.Context, user model.User, password string) (model.Session, error)
}

type Request struct {
	Name                 string `json:"name"`
	Email                string `json:"email"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

func New(service Usecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input Request
		if err := response.Decode(w, r, &input); err != nil {
			response.Error(w, err)
			return
		}
		if input.Password != input.PasswordConfirmation {
			response.Error(w, &model.ValidationError{Field: "password_confirmation", Message: "Пароли не совпадают"})
			return
		}
		auth, err := service.Register(r.Context(), model.User{Name: input.Name, Email: input.Email}, input.Password)
		if err != nil {
			response.Error(w, err)
			return
		}
		session.Set(w, r, auth)
		response.Send(w, http.StatusCreated, response.NewUser(auth.User), nil)
	}
}
