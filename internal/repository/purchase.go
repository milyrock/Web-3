package repository

import (
	"cmp"
	"context"
	"slices"

	"developer.dev/web-3/internal/model"
	sq "github.com/Masterminds/squirrel"
)

func (r *Repository) Purchase(ctx context.Context, items []model.PurchaseItem) error {
	items = slices.SortedFunc(slices.Values(items), func(a, b model.PurchaseItem) int {
		return cmp.Compare(a.ProductID, b.ProductID)
	})
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, item := range items {
		result, err := sq.Update("products").Set("stock", sq.Expr("stock - ?", item.Quantity)).
			Where(sq.Eq{"id": item.ProductID, "available": true}).
			Where(sq.GtOrEq{"stock": item.Quantity}).PlaceholderFormat(sq.Dollar).
			RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed != 1 {
			return model.ErrInsufficientStock
		}
	}
	return tx.Commit()
}
