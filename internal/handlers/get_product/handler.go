package get_product

import (
	"context"
	"net/http"

	"developer.dev/web-3/internal/handlers/response"
	"developer.dev/web-3/internal/model"
)

type Usecase interface {
	Product(ctx context.Context, productID int64) (model.Product, error)
}

func New(service Usecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID, err := response.ID(r)
		if err != nil {
			response.Error(w, err)
			return
		}
		product, err := service.Product(r.Context(), productID)
		response.Send(w, http.StatusOK, response.NewProduct(product), err)
	}
}
