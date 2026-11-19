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
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"developer.dev/web-3/internal/config"
	"developer.dev/web-3/internal/handlers"
	"developer.dev/web-3/internal/handlers/create_feedback"
	"developer.dev/web-3/internal/handlers/login"
	"developer.dev/web-3/internal/handlers/purchase"
	"developer.dev/web-3/internal/handlers/register"
	"developer.dev/web-3/internal/handlers/response"
	"developer.dev/web-3/internal/model"
	"developer.dev/web-3/internal/repository"
	"developer.dev/web-3/internal/usecase"
	"github.com/jackc/pgx/v5"
)

const (
	adminEmail    = "admin@example.com"
	adminPassword = "TestAdminPassword42"
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
	service := usecase.New(repo)
	if err := service.EnsureAdmin(context.Background(), adminEmail, adminPassword); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handlers.New(service, config.Config{WebDir: webDir}))
	t.Cleanup(server.Close)
	return server.URL, testDB
}

func call(t *testing.T, base, method, path string, input any, want int) []byte {
	t.Helper()
	return callAs(t, http.DefaultClient, base, method, path, input, want)
}

func callAs(t *testing.T, client *http.Client, base, method, path string, input any, want int) []byte {
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
	response, err := client.Do(request)
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

func newClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar}
}

func newBuyer(t *testing.T, base string) *http.Client {
	t.Helper()
	client := newClient(t)
	callAs(t, client, base, "POST", "/api/auth/register", register.Request{
		Name: "Test Buyer", Email: "buyer@example.com", Password: "BuyerPassword42", PasswordConfirmation: "BuyerPassword42",
	}, 201)
	return client
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

func TestStaticCaching(t *testing.T) {
	base, _ := setup(t)
	for _, path := range []string{"/", "/account.html", "/script.js?v=2", "/app.css?v=2"} {
		t.Run(path, func(t *testing.T) {
			response, err := http.Get(base + path)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK || response.Header.Get("Cache-Control") != "no-store" {
				t.Fatalf("static response: status=%d, cache=%q", response.StatusCode, response.Header.Get("Cache-Control"))
			}
		})
	}
}

func TestAuthentication(t *testing.T) {
	base, db := setup(t)
	client := newClient(t)
	callAs(t, client, base, "GET", "/api/auth/me", nil, 401)
	input := register.Request{
		Name: "  Test User  ", Email: "CLIENT@EXAMPLE.COM",
		Password: "UserPassword42", PasswordConfirmation: "UserPassword42",
	}
	raw := callAs(t, client, base, "POST", "/api/auth/register", input, 201)
	user := parse[response.User](t, raw)
	if user.ID == 0 || user.Name != "Test User" || user.Email != "client@example.com" || user.Role != "user" {
		t.Fatalf("registered user: %+v", user)
	}
	if fields := parse[map[string]any](t, raw); len(fields) != 4 {
		t.Fatalf("user response fields: %+v", fields)
	}
	var storedPassword string
	if err := db.QueryRow("SELECT password_hash FROM auth_users WHERE id=$1", user.ID).Scan(&storedPassword); err != nil {
		t.Fatal(err)
	}
	if storedPassword == "" || storedPassword == input.Password {
		t.Fatal("password must be hashed")
	}
	current := parse[response.User](t, callAs(t, client, base, "GET", "/api/auth/me", nil, 200))
	if current != user {
		t.Fatalf("current user: %+v", current)
	}
	callAs(t, client, base, "GET", "/api/feedback", nil, 403)
	serverURL, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	cookies := client.Jar.Cookies(serverURL)
	if len(cookies) != 1 {
		t.Fatalf("session cookies: %+v", cookies)
	}
	var storedToken string
	if err := db.QueryRow("SELECT token_hash FROM auth_sessions WHERE user_id=$1", user.ID).Scan(&storedToken); err != nil {
		t.Fatal(err)
	}
	if storedToken == cookies[0].Value {
		t.Fatal("session token must be hashed in storage")
	}
	callAs(t, client, base, "POST", "/api/auth/logout", nil, 204)
	callAs(t, client, base, "GET", "/api/auth/me", nil, 401)
	client.Jar.SetCookies(serverURL, cookies)
	callAs(t, client, base, "GET", "/api/auth/me", nil, 401)
	wrongPassword := call(t, base, "POST", "/api/auth/login",
		login.Request{Email: input.Email, Password: "WrongPassword42"}, 401)
	unknownUser := call(t, base, "POST", "/api/auth/login",
		login.Request{Email: "unknown@example.com", Password: input.Password}, 401)
	if !bytes.Equal(wrongPassword, unknownUser) {
		t.Fatal("login errors should not disclose whether an email exists")
	}
	loggedIn := parse[response.User](t, callAs(t, client, base, "POST", "/api/auth/login",
		login.Request{Email: input.Email, Password: input.Password}, 200))
	if loggedIn != user {
		t.Fatalf("logged in user: %+v", loggedIn)
	}
	if _, err := db.Exec("UPDATE auth_sessions SET expires_at=CURRENT_TIMESTAMP - INTERVAL '1 second' WHERE user_id=$1", user.ID); err != nil {
		t.Fatal(err)
	}
	callAs(t, client, base, "GET", "/api/auth/me", nil, 401)
}

func TestRegistrationValidation(t *testing.T) {
	base, _ := setup(t)
	valid := register.Request{
		Name: "Test User", Email: "user@example.com", Password: "UserPassword42", PasswordConfirmation: "UserPassword42",
	}
	for _, tc := range []struct {
		name string
		edit func(*register.Request)
	}{
		{"name", func(input *register.Request) { input.Name = " " }},
		{"email", func(input *register.Request) { input.Email = "invalid" }},
		{"password", func(input *register.Request) { input.Password, input.PasswordConfirmation = "short", "short" }},
		{"password_confirmation", func(input *register.Request) { input.PasswordConfirmation = "DifferentPassword42" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := valid
			tc.edit(&input)
			out := parse[map[string]string](t, call(t, base, "POST", "/api/auth/register", input, 422))
			if out["field"] != tc.name {
				t.Fatalf("validation error: %+v", out)
			}
		})
	}
	call(t, base, "POST", "/api/auth/register", map[string]any{
		"name": valid.Name, "email": valid.Email, "password": valid.Password,
		"password_confirmation": valid.PasswordConfirmation, "role": "admin",
	}, 422)
	call(t, base, "POST", "/api/auth/register", valid, 201)
	valid.Email = strings.ToUpper(valid.Email)
	call(t, base, "POST", "/api/auth/register", valid, 409)
}

func TestAdminAccess(t *testing.T) {
	base, _ := setup(t)
	call(t, base, "GET", "/api/feedback", nil, 401)
	client := newClient(t)
	user := parse[response.User](t, callAs(t, client, base, "POST", "/api/auth/login",
		login.Request{Email: adminEmail, Password: adminPassword}, 200))
	if user.Email != adminEmail || user.Role != "admin" {
		t.Fatalf("admin: %+v", user)
	}
	input := create_feedback.Request{
		Name: "Test User", Email: "client@example.com", Subject: "Website project",
		Message: "Please help with a website", Consent: true,
	}
	call(t, base, "POST", "/api/feedback", input, 201)
	items := parse[[]response.Feedback](t, callAs(t, client, base, "GET", "/api/feedback", nil, 200))
	if len(items) != 1 || items[0].Subject != input.Subject || items[0].Email != input.Email {
		t.Fatalf("feedback: %+v", items)
	}
}

func TestPurchase(t *testing.T) {
	base, db := setup(t)
	input := purchase.Request{Items: []purchase.Item{{ProductID: 2, Quantity: 2}, {ProductID: 1, Quantity: 3}}}
	call(t, base, "POST", "/api/purchase", input, 401)
	client := newBuyer(t, base)
	out := parse[map[string]string](t, callAs(t, client, base, "POST", "/api/purchase", input, 200))
	if out["message"] == "" {
		t.Fatal("purchase confirmation is missing")
	}
	for productID, want := range map[int64]int{1: 7, 2: 8, 3: 10} {
		var stock int
		if err := db.QueryRow("SELECT stock FROM products WHERE id=$1", productID).Scan(&stock); err != nil {
			t.Fatal(err)
		}
		if stock != want {
			t.Fatalf("product %d stock: got %d, want %d", productID, stock, want)
		}
	}
	product := parse[response.Product](t, call(t, base, "GET", "/api/products/1", nil, 200))
	if product.Stock != 7 {
		t.Fatalf("product stock: %d", product.Stock)
	}
	input.Items = []purchase.Item{{ProductID: 1, Quantity: 7}}
	callAs(t, client, base, "POST", "/api/purchase", input, 200)
	input.Items[0].Quantity = 1
	callAs(t, client, base, "POST", "/api/purchase", input, 409)
	product = parse[response.Product](t, call(t, base, "GET", "/api/products/1", nil, 200))
	if product.Stock != 0 {
		t.Fatalf("sold out product stock: %d", product.Stock)
	}
}

func TestPurchaseRollback(t *testing.T) {
	base, db := setup(t)
	client := newBuyer(t, base)
	if _, err := db.Exec("UPDATE products SET stock=1 WHERE id=2"); err != nil {
		t.Fatal(err)
	}
	input := purchase.Request{Items: []purchase.Item{{ProductID: 1, Quantity: 2}, {ProductID: 2, Quantity: 2}}}
	callAs(t, client, base, "POST", "/api/purchase", input, 409)
	for productID, want := range map[int64]int{1: 10, 2: 1} {
		var stock int
		if err := db.QueryRow("SELECT stock FROM products WHERE id=$1", productID).Scan(&stock); err != nil {
			t.Fatal(err)
		}
		if stock != want {
			t.Fatalf("product %d stock: got %d, want %d", productID, stock, want)
		}
	}
}

func TestPurchaseValidation(t *testing.T) {
	base, _ := setup(t)
	client := newBuyer(t, base)
	for _, tc := range []struct {
		name  string
		items []purchase.Item
	}{
		{"empty", nil},
		{"quantity", []purchase.Item{{ProductID: 1, Quantity: 0}}},
		{"limit", []purchase.Item{{ProductID: 1, Quantity: 11}}},
		{"duplicate", []purchase.Item{{ProductID: 1, Quantity: 1}, {ProductID: 1, Quantity: 1}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			callAs(t, client, base, "POST", "/api/purchase", purchase.Request{Items: tc.items}, 422)
		})
	}
}

func TestConcurrentPurchases(t *testing.T) {
	base, db := setup(t)
	client := newBuyer(t, base)
	if _, err := db.Exec("UPDATE products SET stock=1 WHERE id IN (1,2)"); err != nil {
		t.Fatal(err)
	}
	type result struct {
		status int
		err    error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	for _, items := range [][]purchase.Item{
		{{ProductID: 1, Quantity: 1}, {ProductID: 2, Quantity: 1}},
		{{ProductID: 2, Quantity: 1}, {ProductID: 1, Quantity: 1}},
	} {
		body, err := json.Marshal(purchase.Request{Items: items})
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			<-start
			response, err := client.Post(base+"/api/purchase", "application/json", bytes.NewReader(body))
			if err != nil {
				results <- result{err: err}
				return
			}
			defer response.Body.Close()
			results <- result{status: response.StatusCode}
		}()
	}
	close(start)
	statuses := map[int]int{}
	for range 2 {
		result := <-results
		if result.err != nil {
			t.Fatal(result.err)
		}
		statuses[result.status]++
	}
	if statuses[200] != 1 || statuses[409] != 1 {
		t.Fatalf("purchase statuses: %+v", statuses)
	}
	var stock int
	if err := db.QueryRow("SELECT sum(stock) FROM products WHERE id IN (1,2)").Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if stock != 0 {
		t.Fatalf("remaining stock: %d", stock)
	}
}
