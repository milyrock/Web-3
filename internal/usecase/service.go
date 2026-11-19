package usecase

import (
	"context"
	"time"
	"unicode/utf8"

	"developer.dev/web-3/internal/model"
)

type Repository interface {
	Products(ctx context.Context, pf model.ProductFilter) ([]model.Product, error)
	Product(ctx context.Context, productID int64) (model.Product, error)
	Categories(ctx context.Context) ([]model.Category, error)
	CreateFeedback(ctx context.Context, feedback model.Feedback) (model.Feedback, error)
	Feedback(ctx context.Context) ([]model.Feedback, error)
	CreateUser(ctx context.Context, user model.User) (model.User, error)
	UserByEmail(ctx context.Context, email string) (model.User, error)
	CreateSession(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error
	Session(ctx context.Context, tokenHash string) (model.Session, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	Purchase(ctx context.Context, items []model.PurchaseItem) error
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func invalid(field, message string) error {
	return &model.ValidationError{Field: field, Message: message}
}

func length(text string, min, max int) bool {
	size := utf8.RuneCountInString(text)
	return size >= min && size <= max
}
