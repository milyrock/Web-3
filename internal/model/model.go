package model

import "errors"

var ErrNotFound = errors.New("Запись не найдена")

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
