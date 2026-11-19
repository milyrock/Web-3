package model

import (
	"errors"
	"time"
)

var (
	ErrNotFound          = errors.New("Запись не найдена")
	ErrUnauthorized      = errors.New("Требуется вход")
	ErrCredentials       = errors.New("Неверный email или пароль")
	ErrForbidden         = errors.New("Доступ только для администратора")
	ErrEmailExists       = errors.New("Этот email уже зарегистрирован")
	ErrInsufficientStock = errors.New("Недостаточно услуг в наличии. Проверьте остатки в корзине")
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID           int64
	Name         string
	Email        string
	Role         string
	PasswordHash string
}

type Session struct {
	User      User
	Token     string
	ExpiresAt time.Time
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

type Category struct {
	ID    int64
	Name  string
	Alias string
}

type Property struct {
	ID    int64
	Name  string
	Value string
	Price int64
}

type Image struct {
	ID    int64
	Image string
	Title string
}

type Product struct {
	ID               int64
	ManufacturerID   int64
	CategoryID       int64
	Name             string
	Alias            string
	ShortDescription string
	Description      string
	Price            int64
	Image            string
	Available        bool
	Stock            int
	MetaKeywords     string
	MetaDescription  string
	MetaTitle        string
	Category         string
	Properties       []Property
	Images           []Image
}

type ProductFilter struct {
	Query      string
	CategoryID int64
	Sort       string
}

type PurchaseItem struct {
	ProductID int64
	Quantity  int
}

type Feedback struct {
	ID        int64
	Name      string
	Email     string
	Subject   string
	Message   string
	Rating    *int
	Status    string
	CreatedAt string
}
