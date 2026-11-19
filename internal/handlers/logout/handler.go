package logout

import (
	"context"
	"net/http"

	"developer.dev/web-3/internal/handlers/response"
	"developer.dev/web-3/internal/handlers/session"
)

type Usecase interface {
	Logout(ctx context.Context, token string) error
}

func New(service Usecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := service.Logout(r.Context(), session.Token(r)); err != nil {
			response.Error(w, err)
			return
		}
		session.Clear(w, r)
		w.WriteHeader(http.StatusNoContent)
	}
}
