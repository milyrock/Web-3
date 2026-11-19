package response

import "developer.dev/web-3/internal/model"

type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func NewUser(user model.User) User {
	return User{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role}
}

type Category struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Alias string `json:"alias"`
}

type Property struct {
	ID    int64  `json:"id"`
	Name  string `json:"property_name"`
	Value string `json:"property_value"`
	Price int64  `json:"property_price"`
}

type Image struct {
	ID    int64  `json:"id"`
	Image string `json:"image"`
	Title string `json:"title"`
}

type Product struct {
	ID               int64      `json:"id"`
	ManufacturerID   int64      `json:"manufacturer_id"`
	CategoryID       int64      `json:"category_id"`
	Name             string     `json:"name"`
	Alias            string     `json:"alias"`
	ShortDescription string     `json:"short_description"`
	Description      string     `json:"description"`
	Price            int64      `json:"price"`
	Image            string     `json:"image"`
	Available        bool       `json:"available"`
	Stock            int        `json:"stock"`
	MetaKeywords     string     `json:"meta_keywords"`
	MetaDescription  string     `json:"meta_description"`
	MetaTitle        string     `json:"meta_title"`
	Category         string     `json:"category"`
	Properties       []Property `json:"properties,omitempty"`
	Images           []Image    `json:"images,omitempty"`
}

type Feedback struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
	Rating    *int   `json:"rating"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func NewProduct(product model.Product) Product {
	out := Product{
		ID: product.ID, ManufacturerID: product.ManufacturerID, CategoryID: product.CategoryID,
		Name: product.Name, Alias: product.Alias, ShortDescription: product.ShortDescription,
		Description: product.Description, Price: product.Price, Image: product.Image, Available: product.Available, Stock: product.Stock,
		MetaKeywords: product.MetaKeywords, MetaDescription: product.MetaDescription,
		MetaTitle: product.MetaTitle, Category: product.Category,
	}
	for _, property := range product.Properties {
		out.Properties = append(out.Properties, Property(property))
	}
	for _, image := range product.Images {
		out.Images = append(out.Images, Image(image))
	}
	return out
}
