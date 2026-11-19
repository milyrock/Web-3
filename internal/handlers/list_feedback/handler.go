package list_feedback

import (
	"context"
	"net/http"

	"developer.dev/web-3/internal/handlers/response"
	"developer.dev/web-3/internal/handlers/session"
	"developer.dev/web-3/internal/model"
)

type Usecase interface {
	Feedback(ctx context.Context, token string) ([]model.Feedback, error)
}

func New(service Usecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := service.Feedback(r.Context(), session.Token(r))
		out := make([]response.Feedback, len(items))
		for i, feedback := range items {
			out[i] = response.Feedback(feedback)
		}
		response.Send(w, http.StatusOK, out, err)
	}
}
