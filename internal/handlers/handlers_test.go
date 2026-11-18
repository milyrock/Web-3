package handlers_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"developer.dev/web-3/internal/config"
	"developer.dev/web-3/internal/handlers"
	"developer.dev/web-3/internal/handlers/create_feedback"
	"developer.dev/web-3/internal/handlers/response"
	"developer.dev/web-3/internal/model"
	"developer.dev/web-3/internal/repository"
	"developer.dev/web-3/internal/usecase"
	"github.com/jackc/pgx/v5"
)

func setup(t *testing.T) (string, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run tests against PostgreSQL")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	schema := "lab_test_" + hex.EncodeToString(raw)
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := db.Exec("CREATE SCHEMA " + identifier); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("DROP SCHEMA " + identifier + " CASCADE"); err != nil {
			t.Error(err)
		}
		db.Close()
	})
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	repo, err := repository.Open(context.Background(), u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	testDB, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testDB.Close() })
	webDir, err := filepath.Abs("../../web")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handlers.New(usecase.New(repo), config.Config{WebDir: webDir}))
	t.Cleanup(server.Close)
	return server.URL, testDB
}

func call(t *testing.T, base, method, path string, input any, want int) []byte {
	t.Helper()
	var body io.Reader
	if input != nil {
		raw, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(raw)
	}
	request, err := http.NewRequest(method, base+path, body)
	if err != nil {
		t.Fatal(err)
	}
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != want {
		t.Fatalf("%s %s: got %d, want %d: %s", method, path, response.StatusCode, want, raw)
	}
	return raw
}

func parse[T any](t *testing.T, raw []byte) T {
	t.Helper()
	var result T
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestCatalog(t *testing.T) {
	base, _ := setup(t)
	categories := parse[[]response.Category](t, call(t, base, "GET", "/api/categories", nil, 200))
	if len(categories) != 3 {
		t.Fatalf("categories: %+v", categories)
	}
	for _, tc := range []struct {
		sort string
		ids  []int64
	}{
		{"price_asc", []int64{6, 4, 3, 1, 5, 2}},
		{"price_desc", []int64{2, 5, 1, 3, 4, 6}},
		{"newest", []int64{6, 5, 4, 3, 2, 1}},
	} {
		t.Run(tc.sort, func(t *testing.T) {
			products := parse[[]response.Product](t, call(t, base, "GET", "/api/products?sort="+tc.sort, nil, 200))
			if len(products) != len(tc.ids) {
				t.Fatalf("products: %+v", products)
			}
			for i, product := range products {
				if product.ID != tc.ids[i] {
					t.Fatalf("product at position %d: got %d, want %d", i, product.ID, tc.ids[i])
				}
			}
		})
	}
	found := parse[[]response.Product](t, call(t, base, "GET", "/api/products?q="+url.QueryEscape("САЙТ")+"&category=1", nil, 200))
	if len(found) == 0 {
		t.Fatal("search returned no products")
	}
	for _, product := range found {
		if product.CategoryID != 1 {
			t.Fatalf("category filter: %+v", product)
		}
	}
	product := parse[response.Product](t, call(t, base, "GET", "/api/products/1", nil, 200))
	if len(product.Properties) != 1 || len(product.Images) != 1 {
		t.Fatal("product details are missing")
	}
	if product.CategoryID != 1 || product.ShortDescription == "" ||
		product.Properties[0].Name == "" || product.Properties[0].Value == "" {
		t.Fatalf("product details: %+v", product)
	}
	call(t, base, "GET", "/api/products/999999", nil, 404)
	call(t, base, "GET", "/api/products?q="+url.QueryEscape(strings.Repeat("a", 101)), nil, 422)
	call(t, base, "GET", "/api/products?sort=invalid", nil, 422)
	call(t, base, "GET", "/api/products?category=invalid", nil, 422)
}

func TestAvailableProducts(t *testing.T) {
	base, db := setup(t)
	if _, err := db.Exec("UPDATE products SET available=FALSE WHERE id=$1", 1); err != nil {
		t.Fatal(err)
	}
	products := parse[[]response.Product](t, call(t, base, "GET", "/api/products", nil, 200))
	if len(products) != 5 {
		t.Fatalf("products: %+v", products)
	}
	for _, product := range products {
		if !product.Available {
			t.Fatalf("unavailable product: %+v", product)
		}
	}
	call(t, base, "GET", "/api/products/1", nil, 404)
}

func TestFeedback(t *testing.T) {
	base, db := setup(t)
	input := create_feedback.Request{
		Name: "Test User", Email: "CLIENT@EXAMPLE.COM", Subject: "Project discussion",
		Message: "Please help with a new website", Consent: true,
	}
	created := parse[response.Feedback](t, call(t, base, "POST", "/api/feedback", input, 201))
	if created.ID == 0 || created.CreatedAt == "" || created.Status != "new" ||
		created.Email != "client@example.com" || created.Rating != nil {
		t.Fatalf("feedback: %+v", created)
	}
	var saved model.Feedback
	if err := db.QueryRow("SELECT name,email,subject,message,rating FROM feedback WHERE id=$1", created.ID).
		Scan(&saved.Name, &saved.Email, &saved.Subject, &saved.Message, &saved.Rating); err != nil {
		t.Fatal(err)
	}
	if saved.Name != input.Name || saved.Email != "client@example.com" ||
		saved.Subject != input.Subject || saved.Message != input.Message || saved.Rating != nil {
		t.Fatalf("saved feedback: %+v", saved)
	}
}

func TestFeedbackValidation(t *testing.T) {
	base, _ := setup(t)
	valid := create_feedback.Request{
		Name: "Test User", Email: "test@example.com", Subject: "Project discussion",
		Message: "Please help with a new website", Consent: true,
	}
	cases := []struct {
		name string
		edit func(*create_feedback.Request)
	}{
		{"name", func(input *create_feedback.Request) { input.Name = " " }},
		{"email", func(input *create_feedback.Request) { input.Email = "invalid" }},
		{"subject", func(input *create_feedback.Request) { input.Subject = "" }},
		{"message", func(input *create_feedback.Request) { input.Message = "Short" }},
		{"rating", func(input *create_feedback.Request) { rating := 6; input.Rating = &rating }},
		{"consent", func(input *create_feedback.Request) { input.Consent = false }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := valid
			tc.edit(&input)
			out := parse[map[string]string](t, call(t, base, "POST", "/api/feedback", input, 422))
			if out["field"] != tc.name || out["error"] == "" {
				t.Fatalf("validation error: %+v", out)
			}
		})
	}
}

func TestBrowserSecurity(t *testing.T) {
	base, _ := setup(t)
	request, err := http.NewRequest("POST", base+"/api/feedback", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", "https://other.example")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatalf("cross-origin request: %d", response.StatusCode)
	}
	call(t, base, "GET", "/assets/", nil, 404)
	call(t, base, "GET", "/../go.mod", nil, 404)
	call(t, base, "GET", "/services.html", nil, 200)
}
