package usecase

import (
	"context"
	"strings"

	"developer.dev/web-3/internal/model"
)

func (s *Service) Products(ctx context.Context, pf model.ProductFilter) ([]model.Product, error) {
	pf.Query = strings.TrimSpace(pf.Query)
	if !length(pf.Query, 0, 100) {
		return nil, invalid("q", "Поисковый запрос должен быть не длиннее 100 символов")
	}
	if pf.CategoryID < 0 {
		return nil, invalid("category", "Некорректная категория")
	}
	switch pf.Sort {
	case "", "name", "price_asc", "price_desc", "newest":
	default:
		return nil, invalid("sort", "Неизвестный способ сортировки")
	}
	return s.repo.Products(ctx, pf)
}

func (s *Service) Product(ctx context.Context, productID int64) (model.Product, error) {
	return s.repo.Product(ctx, productID)
}

func (s *Service) Categories(ctx context.Context) ([]model.Category, error) {
	return s.repo.Categories(ctx)
}
