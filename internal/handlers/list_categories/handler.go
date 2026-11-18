package list_categories

import (
	"context"
	"net/http"

	"developer.dev/web-3/internal/handlers/response"
	"developer.dev/web-3/internal/model"
)

type Usecase interface {
	Categories(ctx context.Context) ([]model.Category, error)
}

func New(service Usecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := service.Categories(r.Context())
		out := make([]response.Category, len(items))
		for i, category := range items {
			out[i] = response.Category(category)
		}
		response.Send(w, http.StatusOK, out, err)
	}
}
