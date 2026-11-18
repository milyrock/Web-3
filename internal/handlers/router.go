package handlers

import (
	"net/http"
	"strings"

	"developer.dev/web-3/internal/config"
	"developer.dev/web-3/internal/handlers/create_feedback"
	"developer.dev/web-3/internal/handlers/get_product"
	"developer.dev/web-3/internal/handlers/list_categories"
	"developer.dev/web-3/internal/handlers/list_products"
	"developer.dev/web-3/internal/handlers/static"
)

type Usecase interface {
	list_categories.Usecase
	list_products.Usecase
	get_product.Usecase
	create_feedback.Usecase
}

func New(service Usecase, cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/categories", list_categories.New(service))
	mux.Handle("GET /api/products", list_products.New(service))
	mux.Handle("GET /api/products/{id}", get_product.New(service))
	mux.Handle("POST /api/feedback", create_feedback.New(service))
	mux.Handle("/", static.New(cfg.WebDir))
	protected := http.NewCrossOriginProtection().Handler(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; font-src 'self'; script-src 'self'; frame-src https://maps.google.com; connect-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		protected.ServeHTTP(w, r)
	})
}
