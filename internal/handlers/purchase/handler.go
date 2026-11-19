package purchase

import (
	"context"
	"net/http"

	"developer.dev/web-3/internal/handlers/response"
	"developer.dev/web-3/internal/handlers/session"
	"developer.dev/web-3/internal/model"
)

type Usecase interface {
	Purchase(ctx context.Context, token string, items []model.PurchaseItem) error
}

type Item struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type Request struct {
	Items []Item `json:"items"`
}

func New(service Usecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input Request
		if err := response.Decode(w, r, &input); err != nil {
			response.Error(w, err)
			return
		}
		items := make([]model.PurchaseItem, len(input.Items))
		for i, item := range input.Items {
			items[i] = model.PurchaseItem{ProductID: item.ProductID, Quantity: item.Quantity}
		}
		err := service.Purchase(r.Context(), session.Token(r), items)
		response.Send(w, http.StatusOK, map[string]string{"message": "Успех! Покупка выполнена"}, err)
	}
}
