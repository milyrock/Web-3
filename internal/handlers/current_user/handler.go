package current_user

import (
	"context"
	"net/http"

	"developer.dev/web-3/internal/handlers/response"
	"developer.dev/web-3/internal/handlers/session"
	"developer.dev/web-3/internal/model"
)

type Usecase interface {
	CurrentUser(ctx context.Context, token string) (model.User, error)
}

func New(service Usecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := service.CurrentUser(r.Context(), session.Token(r))
		response.Send(w, http.StatusOK, response.NewUser(user), err)
	}
}
