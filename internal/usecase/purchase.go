package usecase

import (
	"context"

	"developer.dev/web-3/internal/model"
)

func (s *Service) Purchase(ctx context.Context, token string, items []model.PurchaseItem) error {
	if _, err := s.CurrentUser(ctx, token); err != nil {
		return err
	}
	if len(items) == 0 || len(items) > 50 {
		return invalid("items", "В корзине должно быть от 1 до 50 услуг")
	}
	seen := make(map[int64]bool, len(items))
	for _, item := range items {
		if item.ProductID < 1 || item.Quantity < 1 || item.Quantity > 10 || seen[item.ProductID] {
			return invalid("items", "Проверьте услуги и количество от 1 до 10 без повторов")
		}
		seen[item.ProductID] = true
	}
	return s.repo.Purchase(ctx, items)
}
