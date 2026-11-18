package list_products

import (
	"context"
	"net/http"

	"developer.dev/web-3/internal/handlers/response"
	"developer.dev/web-3/internal/model"
)

type Usecase interface {
	Products(ctx context.Context, pf model.ProductFilter) ([]model.Product, error)
}

func New(service Usecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter, err := response.ProductFilter(r)
		if err != nil {
			response.Error(w, err)
			return
		}
		items, err := service.Products(r.Context(), filter)
		out := make([]response.Product, len(items))
		for i, product := range items {
			out[i] = response.NewProduct(product)
		}
		response.Send(w, http.StatusOK, out, err)
	}
}
