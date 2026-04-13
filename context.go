package flux

import (
	"bytes"
	"net/http"
)

// Context holds all the information for an HTTP request/response cycle.
type Context struct {
	Writer     http.ResponseWriter
	Request    *http.Request
	app        *Flux
	params     []Param
	store      map[string]any // per-request key/value store
	statusCode int            // tracked for Logger middleware
	written    bool           // true once a response has been committed
}

func (c *Context) reset() {
	c.Writer = nil
	c.Request = nil
	c.statusCode = 0
	c.written = false
	c.params = c.params[:0]
	if c.store != nil {
		for k := range c.store {
			delete(c.store, k)
		}
	}
}

// Param represents a URL path parameter (e.g. :id → {Key:"id", Value:"123"}).
type Param struct {
	Key   string
	Value string
}

// Param returns the value of the named path parameter, or "" if not present.
func (c *Context) Param(name string) string {
	for _, p := range c.params {
		if p.Key == name {
			return p.Value
		}
	}
	return ""
}

// Query returns the first value of the named URL query parameter.
func (c *Context) Query(name string) string {
	return c.Request.URL.Query().Get(name)
}

// QueryDefault returns the named query parameter, or defaultValue if absent.
func (c *Context) QueryDefault(name, defaultValue string) string {
	if val := c.Query(name); val != "" {
		return val
	}
	return defaultValue
}

// Header returns the value of the named request header.
func (c *Context) Header(name string) string {
	return c.Request.Header.Get(name)
}

// SetHeader sets a response header key/value pair.
func (c *Context) SetHeader(key, value string) {
	c.Writer.Header().Set(key, value)
}

// SetContentType is a convenience wrapper for SetHeader("Content-Type", ...).
func (c *Context) SetContentType(contentType string) {
	c.Writer.Header().Set("Content-Type", contentType)
}

// JSON marshals data to JSON and writes it with the given HTTP status code.
// After the first call, subsequent calls within the same request are no-ops
// to prevent double-write panics.
func (c *Context) JSON(code int, data any) error {
	if c.written {
		return nil
	}

	b, err := c.app.encoder.Marshal(data)
	if err != nil {
		return err
	}

	c.Writer.Header().Set(headerContentType, headerValueJSON)
	c.Writer.WriteHeader(code)
	c.statusCode = code
	c.written = true
	_, err = c.Writer.Write(b)
	return err
}

const (
	headerContentType = "Content-Type"
	headerValueJSON   = "application/json"
	headerValueHTML   = "text/html; charset=utf-8"
	headerValuePlain  = "text/plain; charset=utf-8"
)

// StatusJSON is an alias for JSON (for explicit status-code readability).
func (c *Context) StatusJSON(code int, data any) error {
	return c.JSON(code, data)
}

// String writes a plain-text response with the given HTTP status code.
func (c *Context) String(code int, text string) error {
	if c.written {
		return nil
	}
	c.Writer.Header().Set(headerContentType, headerValuePlain)
	c.Writer.WriteHeader(code)
	c.statusCode = code
	c.written = true
	_, err := c.Writer.Write([]byte(text))
	return err
}

// HTML writes an HTML response with the given HTTP status code.
func (c *Context) HTML(code int, html string) error {
	if c.written {
		return nil
	}
	c.Writer.Header().Set(headerContentType, headerValueHTML)
	c.Writer.WriteHeader(code)
	c.statusCode = code
	c.written = true
	_, err := c.Writer.Write([]byte(html))
	return err
}

// BindJSON reads the request body, JSON-decodes it into v using the configured
// encoder, and runs validation. The body is always closed after this call.
func (c *Context) BindJSON(v any) error {
	defer c.Request.Body.Close()

	buf := c.app.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer c.app.bufPool.Put(buf)

	if _, err := buf.ReadFrom(c.Request.Body); err != nil {
		return NewHTTPError(http.StatusBadRequest, "Failed to read request body", err.Error())
	}

	if err := c.app.encoder.Unmarshal(buf.Bytes(), v); err != nil {
		return NewHTTPError(http.StatusBadRequest, "Invalid JSON", err.Error())
	}

	if err := NewValidator().Validate(v); err != nil {
		return NewHTTPError(http.StatusUnprocessableEntity, "Validation failed", err.Error())
	}
	return nil
}

// Body reads and returns the raw request body. The caller is responsible for
// closing the body.
func (c *Context) Body() ([]byte, error) {
	defer c.Request.Body.Close()

	buf := c.app.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer c.app.bufPool.Put(buf)

	if _, err := buf.ReadFrom(c.Request.Body); err != nil {
		return nil, err
	}

	// Note: We return a copy here because the pooled buffer will be reused.
	// For zero-allocation raw body access, we'd need a different pattern.
	data := make([]byte, buf.Len())
	copy(data, buf.Bytes())
	return data, nil
}

// Method returns the HTTP method of the current request.
func (c *Context) Method() string {
	return c.Request.Method
}

// Path returns the URL path of the current request.
func (c *Context) Path() string {
	return c.Request.URL.Path
}

// Set stores an arbitrary value in the per-request context store.
func (c *Context) Set(key string, value any) {
	if c.store == nil {
		c.store = make(map[string]any)
	}
	c.store[key] = value
}

// Get retrieves a value from the per-request context store.
func (c *Context) Get(key string) (any, bool) {
	if c.store == nil {
		return nil, false
	}
	val, ok := c.store[key]
	return val, ok
}

// MustGet retrieves a value from the context store, panicking if not found.
func (c *Context) MustGet(key string) any {
	val, ok := c.Get(key)
	if !ok {
		panic("flux: key not found in context store: " + key)
	}
	return val
}

// Cookie returns the value of the named cookie, or "" if not present.
func (c *Context) Cookie(name string) string {
	cookie, err := c.Request.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// SetCookie adds a Set-Cookie header to the response.
func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Writer, cookie)
}

// Redirect sends an HTTP redirect to url with the given status code.
func (c *Context) Redirect(code int, url string) error {
	http.Redirect(c.Writer, c.Request, url, code)
	c.statusCode = code
	c.written = true
	return nil
}

// NoContent sends a 204 No Content response.
func (c *Context) NoContent() error {
	if c.written {
		return nil
	}
	c.Writer.WriteHeader(http.StatusNoContent)
	c.statusCode = http.StatusNoContent
	c.written = true
	return nil
}
