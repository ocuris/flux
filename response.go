package flux

import "net/http"

// The following methods on Context are convenience shortcuts for the most
// common HTTP response patterns. They wrap c.JSON / NewHTTPError so handler
// code stays lean and self-documenting.

// OK sends a 200 OK JSON response.
func (c *Context) OK(data any) error {
	return c.JSON(http.StatusOK, data)
}

// Created sends a 201 Created JSON response.
func (c *Context) Created(data any) error {
	return c.JSON(http.StatusCreated, data)
}

// Accepted sends a 202 Accepted JSON response.
func (c *Context) Accepted(data any) error {
	return c.JSON(http.StatusAccepted, data)
}

// BadRequest returns a 400 Bad Request HTTPError.
func (c *Context) BadRequest(message string, details ...any) error {
	return NewHTTPError(http.StatusBadRequest, message, details...)
}

// Unauthorized returns a 401 Unauthorized HTTPError.
func (c *Context) Unauthorized(message string) error {
	return NewHTTPError(http.StatusUnauthorized, message)
}

// Forbidden returns a 403 Forbidden HTTPError.
func (c *Context) Forbidden(message string) error {
	return NewHTTPError(http.StatusForbidden, message)
}

// NotFound returns a 404 Not Found HTTPError.
func (c *Context) NotFound(message string) error {
	return NewHTTPError(http.StatusNotFound, message)
}

// InternalServerError returns a 500 Internal Server Error HTTPError.
func (c *Context) InternalServerError(message string) error {
	return NewHTTPError(http.StatusInternalServerError, message)
}
