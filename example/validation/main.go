package main

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/ocuris/flux"
)

// ── Domain types ──────────────────────────────────────────────────────────────

// Product is the catalog domain model.
type Product struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	SKU      string  `json:"sku"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
	Stock    int     `json:"stock"`
}

// CreateProductRequest is the payload for POST /products.
type CreateProductRequest struct {
	Name     string  `json:"name"`
	SKU      string  `json:"sku"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
	Stock    int     `json:"stock"`
}

// UpdateProductRequest is the payload for PUT /products/:id.
// Pointer fields let callers omit fields they do not wish to update.
type UpdateProductRequest struct {
	Name     *string  `json:"name,omitempty"`
	Price    *float64 `json:"price,omitempty"`
	Category *string  `json:"category,omitempty"`
	Stock    *int     `json:"stock,omitempty"`
}

// ValidationError is a field-level validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationResponse wraps one or more field errors.
type ValidationResponse struct {
	Errors []ValidationError `json:"errors"`
}

// ── Validation helpers ────────────────────────────────────────────────────────

var (
	skuRegex          = regexp.MustCompile(`^[A-Z0-9\-]{3,20}$`)
	allowedCategories = []string{"electronics", "clothing", "food", "books", "other"}
)

func validateProduct(req CreateProductRequest) []ValidationError {
	var errs []ValidationError

	name := strings.TrimSpace(req.Name)
	if name == "" {
		errs = append(errs, ValidationError{"name", "Name is required"})
	} else if len(name) < 3 || len(name) > 100 {
		errs = append(errs, ValidationError{"name", "Name must be 3–100 characters"})
	}

	sku := strings.ToUpper(strings.TrimSpace(req.SKU))
	if sku == "" {
		errs = append(errs, ValidationError{"sku", "SKU is required"})
	} else if !skuRegex.MatchString(sku) {
		errs = append(errs, ValidationError{"sku", "SKU must be 3–20 uppercase alphanumeric characters or hyphens"})
	}

	if req.Price <= 0 {
		errs = append(errs, ValidationError{"price", "Price must be greater than 0"})
	} else if req.Price > 999_999.99 {
		errs = append(errs, ValidationError{"price", "Price exceeds the maximum allowed value (999999.99)"})
	}

	cat := strings.TrimSpace(req.Category)
	if cat == "" {
		errs = append(errs, ValidationError{"category", "Category is required"})
	} else if !containsStr(allowedCategories, cat) {
		errs = append(errs, ValidationError{"category", "Allowed categories: electronics, clothing, food, books, other"})
	}

	if req.Stock < 0 {
		errs = append(errs, ValidationError{"stock", "Stock cannot be negative"})
	}

	return errs
}

func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ── Thread-safe in-memory store ───────────────────────────────────────────────

type productStore struct {
	mu       sync.RWMutex
	products map[int]*Product
	nextID   int
}

func newProductStore() *productStore {
	return &productStore{products: make(map[int]*Product), nextID: 1}
}

func (s *productStore) list(page, limit int) ([]*Product, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := make([]*Product, 0, len(s.products))
	for _, p := range s.products {
		all = append(all, p)
	}
	total := len(all)
	// Simple offset pagination
	start := (page - 1) * limit
	if start >= total {
		return []*Product{}, total
	}
	end := start + limit
	if end > total {
		end = total
	}
	return all[start:end], total
}

func (s *productStore) get(id int) (*Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	return p, ok
}

func (s *productStore) create(req CreateProductRequest) *Product {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := &Product{
		ID:       s.nextID,
		Name:     strings.TrimSpace(req.Name),
		SKU:      strings.ToUpper(strings.TrimSpace(req.SKU)),
		Price:    req.Price,
		Category: strings.TrimSpace(req.Category),
		Stock:    req.Stock,
	}
	s.products[s.nextID] = p
	s.nextID++
	return p
}

func (s *productStore) update(id int, req UpdateProductRequest) (*Product, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.products[id]
	if !ok {
		return nil, false
	}
	if req.Name != nil {
		p.Name = strings.TrimSpace(*req.Name)
	}
	if req.Price != nil {
		p.Price = *req.Price
	}
	if req.Category != nil {
		p.Category = strings.TrimSpace(*req.Category)
	}
	if req.Stock != nil {
		p.Stock = *req.Stock
	}
	return p, true
}

func (s *productStore) delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.products[id]; !ok {
		return false
	}
	delete(s.products, id)
	return true
}

var store = newProductStore()

// ── Application entry point ───────────────────────────────────────────────────

func main() {
	app := flux.New(flux.Config{
		Title:       "Flux Validation Example",
		Version:     "1.0.0",
		Description: "Demonstrates comprehensive input validation patterns with structured error responses.",
		Debug:       false,
	})

	app.Use(flux.Recover())
	app.Use(flux.RequestID())
	app.Use(flux.SecurityHeaders())

	app.GET("/products", flux.Doc(
		"List Products",
		"Returns a paginated list of products.",
		"products",
	).
		Param("page", "query", "Page number (default 1)", "integer", false).
		Param("limit", "query", "Items per page, max 100 (default 20)", "integer", false).
		Response(200, "Paginated product list", "application/json", nil),
		listProducts,
	)

	app.GET("/products/:id", flux.Doc(
		"Get Product",
		"Returns a single product by numeric ID.",
		"products",
	).
		Param("id", "path", "Numeric product ID", "integer", true).
		Response(200, "Product object", "application/json", nil).
		Response(404, "Product not found", "application/json", nil),
		getProduct,
	)

	app.POST("/products", flux.Doc(
		"Create Product",
		"Creates a new product. All fields are validated before insertion.",
		"products",
	).
		RequestBody("Product data", CreateProductRequest{
			Name: "Wireless Headphones", SKU: "WHD-001", Price: 79.99,
			Category: "electronics", Stock: 50,
		}).
		Response(201, "Created product", "application/json", nil).
		Response(400, "Validation errors", "application/json", nil),
		createProduct,
	)

	app.PUT("/products/:id", flux.Doc(
		"Update Product",
		"Partially updates a product. Only provided (non-null) fields are applied.",
		"products",
	).
		Param("id", "path", "Numeric product ID", "integer", true).
		RequestBody("Fields to update", UpdateProductRequest{}).
		Response(200, "Updated product", "application/json", nil).
		Response(400, "Validation error", "application/json", nil).
		Response(404, "Product not found", "application/json", nil),
		updateProduct,
	)

	app.DELETE("/products/:id", flux.Doc(
		"Delete Product",
		"Permanently removes a product.",
		"products",
	).
		Param("id", "path", "Numeric product ID", "integer", true).
		Response(204, "Deleted — no body", "application/json", nil).
		Response(404, "Product not found", "application/json", nil),
		deleteProduct,
	)

	if err := app.Start(":8002"); err != nil {
		panic(err)
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func listProducts(c *flux.Context) error {
	page, _ := strconv.Atoi(c.QueryDefault("page", "1"))
	limit, _ := strconv.Atoi(c.QueryDefault("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	products, total := store.list(page, limit)
	return c.JSON(200, flux.Map{
		"products": products,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

func getProduct(c *flux.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return flux.NewHTTPError(400, "Invalid product ID — must be a positive integer")
	}

	product, ok := store.get(id)
	if !ok {
		return flux.NewHTTPError(404, "Product not found")
	}

	return c.JSON(200, product)
}

func createProduct(c *flux.Context) error {
	var req CreateProductRequest
	if err := c.BindJSON(&req); err != nil {
		return err
	}

	if errs := validateProduct(req); len(errs) > 0 {
		// Return 422 with field-level error details, not a generic 400.
		return c.JSON(422, ValidationResponse{Errors: errs})
	}

	return c.JSON(201, store.create(req))
}

func updateProduct(c *flux.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return flux.NewHTTPError(400, "Invalid product ID — must be a positive integer")
	}

	var req UpdateProductRequest
	if err := c.BindJSON(&req); err != nil {
		return err
	}

	// Validate only the fields that were provided
	if req.Name != nil && (len(*req.Name) < 3 || len(*req.Name) > 100) {
		return flux.NewHTTPError(400, "Name must be 3–100 characters")
	}
	if req.Price != nil && *req.Price <= 0 {
		return flux.NewHTTPError(400, "Price must be greater than 0")
	}
	if req.Stock != nil && *req.Stock < 0 {
		return flux.NewHTTPError(400, "Stock cannot be negative")
	}
	if req.Category != nil && !containsStr(allowedCategories, *req.Category) {
		return flux.NewHTTPError(400, "Invalid category")
	}

	product, ok := store.update(id, req)
	if !ok {
		return flux.NewHTTPError(404, "Product not found")
	}

	return c.JSON(200, product)
}

func deleteProduct(c *flux.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return flux.NewHTTPError(400, "Invalid product ID — must be a positive integer")
	}

	if !store.delete(id) {
		return flux.NewHTTPError(404, "Product not found")
	}

	return c.NoContent()
}
