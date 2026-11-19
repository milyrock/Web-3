package repository

import (
	"context"
	"database/sql"
	"errors"

	"developer.dev/web-3/internal/model"
	sq "github.com/Masterminds/squirrel"
)

func scanProduct(row sq.RowScanner) (model.Product, error) {
	var product model.Product
	err := row.Scan(&product.ID, &product.ManufacturerID, &product.CategoryID, &product.Name, &product.Alias, &product.ShortDescription,
		&product.Description, &product.Price, &product.Image, &product.Available, &product.Stock, &product.MetaKeywords, &product.MetaDescription, &product.MetaTitle, &product.Category)
	return product, err
}

func (r *Repository) Products(ctx context.Context, pf model.ProductFilter) ([]model.Product, error) {
	query := sq.Select(
		"p.id", "p.manufacturer_id", "p.category_id", "p.name", "p.alias",
		"p.short_description", "p.description", "p.price", "p.image", "p.available", "p.stock",
		"p.meta_keywords", "p.meta_description", "p.meta_title", "c.name",
	).From("products p").Join("categories c ON c.id = p.category_id").
		Where(sq.Eq{"p.available": true}).PlaceholderFormat(sq.Dollar).RunWith(r.db)
	if pf.Query != "" {
		query = query.Where(sq.Or{
			sq.Expr("strpos(lower(p.name), lower(?)) > 0", pf.Query),
			sq.Expr("strpos(lower(p.description), lower(?)) > 0", pf.Query),
			sq.Expr("strpos(lower(p.short_description), lower(?)) > 0", pf.Query),
		})
	}
	if pf.CategoryID != 0 {
		query = query.Where(sq.Eq{"p.category_id": pf.CategoryID})
	}
	switch pf.Sort {
	case "name":
		query = query.OrderBy("p.name", "p.id")
	case "price_asc":
		query = query.OrderBy("p.price", "p.id")
	case "price_desc":
		query = query.OrderBy("p.price DESC", "p.id")
	case "newest":
		query = query.OrderBy("p.id DESC")
	default:
		query = query.OrderBy("p.id")
	}
	rows, err := query.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.Product{}
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, product)
	}
	return items, rows.Err()
}

func (r *Repository) Product(ctx context.Context, productID int64) (model.Product, error) {
	product, err := scanProduct(sq.Select(
		"p.id", "p.manufacturer_id", "p.category_id", "p.name", "p.alias",
		"p.short_description", "p.description", "p.price", "p.image", "p.available", "p.stock",
		"p.meta_keywords", "p.meta_description", "p.meta_title", "c.name",
	).From("products p").Join("categories c ON c.id = p.category_id").
		Where(sq.Eq{"p.id": productID, "p.available": true}).
		PlaceholderFormat(sq.Dollar).RunWith(r.db).QueryRowContext(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return product, model.ErrNotFound
	}
	if err != nil {
		return product, err
	}
	rows, err := sq.Select("id", "property_name", "property_value", "property_price").
		From("product_properties").Where(sq.Eq{"product_id": productID}).
		OrderBy("id").PlaceholderFormat(sq.Dollar).RunWith(r.db).QueryContext(ctx)
	if err != nil {
		return product, err
	}
	for rows.Next() {
		var prop model.Property
		if err := rows.Scan(&prop.ID, &prop.Name, &prop.Value, &prop.Price); err != nil {
			rows.Close()
			return product, err
		}
		product.Properties = append(product.Properties, prop)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return product, err
	}
	rows, err = sq.Select("id", "image", "title").From("product_images").
		Where(sq.Eq{"product_id": productID}).OrderBy("id").
		PlaceholderFormat(sq.Dollar).RunWith(r.db).QueryContext(ctx)
	if err != nil {
		return product, err
	}
	defer rows.Close()
	for rows.Next() {
		var image model.Image
		if err := rows.Scan(&image.ID, &image.Image, &image.Title); err != nil {
			return product, err
		}
		product.Images = append(product.Images, image)
	}
	return product, rows.Err()
}

func (r *Repository) Categories(ctx context.Context) ([]model.Category, error) {
	rows, err := sq.Select("id", "name", "alias").From("categories").
		OrderBy("id").PlaceholderFormat(sq.Dollar).RunWith(r.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.Category{}
	for rows.Next() {
		var category model.Category
		if err := rows.Scan(&category.ID, &category.Name, &category.Alias); err != nil {
			return nil, err
		}
		items = append(items, category)
	}
	return items, rows.Err()
}
