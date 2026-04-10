package flux

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig holds the configuration for the JWT authentication middleware.
type JWTConfig struct {
	// SecretKey is the HMAC signing secret used to validate tokens.
	SecretKey []byte

	// Skipper is an optional function; return true to bypass JWT validation
	// for a given request (e.g. public health-check endpoints).
	Skipper func(c *Context) bool

	// ContextKey is the key under which the parsed *jwt.Token is stored in
	// the request context. Defaults to "user".
	ContextKey string

	// TokenLookup defines how to extract the raw token string.
	// Format: "source:name"  where source is one of: header, query, cookie.
	// Default: "header:Authorization"  (supports "Bearer <token>" prefix)
	TokenLookup string
}

// JWT returns middleware that validates HMAC-signed JWT tokens and stores the
// parsed token in the context under config.ContextKey.
//
//	app.Use(flux.JWT(flux.JWTConfig{SecretKey: []byte("my-secret")}))
//
// Access the token inside a handler:
//
//	token  := c.MustGet("user").(*jwt.Token)
//	claims := token.Claims.(jwt.MapClaims)
func JWT(config JWTConfig) MiddlewareFunc {
	if config.ContextKey == "" {
		config.ContextKey = "user"
	}
	if config.TokenLookup == "" {
		config.TokenLookup = "header:Authorization"
	}

	parts := strings.SplitN(config.TokenLookup, ":", 2)
	if len(parts) != 2 {
		panic("flux: invalid JWTConfig.TokenLookup — expected format 'source:name'")
	}
	source, name := parts[0], parts[1]

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			if config.Skipper != nil && config.Skipper(c) {
				return next(c)
			}

			raw := extractToken(c, source, name)
			if raw == "" {
				return NewHTTPError(http.StatusUnauthorized, "missing or malformed token")
			}

			token, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
				// Reject tokens not signed with an HMAC algorithm
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return config.SecretKey, nil
			})
			if err != nil || !token.Valid {
				return NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			c.Set(config.ContextKey, token)
			return next(c)
		}
	}
}

// JWTClaims extracts jwt.MapClaims from a previously validated token stored
// in the context. It panics if the token is absent or has unexpected type.
//
//	claims := flux.JWTClaims(c)
//	email  := claims["email"].(string)
func JWTClaims(c *Context, contextKey ...string) jwt.MapClaims {
	key := "user"
	if len(contextKey) > 0 {
		key = contextKey[0]
	}
	token, ok := c.MustGet(key).(*jwt.Token)
	if !ok {
		panic("flux: value stored at '" + key + "' is not a *jwt.Token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		panic("flux: JWT claims are not jwt.MapClaims")
	}
	return claims
}

// extractToken pulls the raw JWT string from the request based on source.
func extractToken(c *Context, source, name string) string {
	switch source {
	case "header":
		val := c.Header(name)
		// "Authorization: Bearer <token>" is the standard format
		if strings.EqualFold(name, "authorization") {
			if after, found := strings.CutPrefix(val, "Bearer "); found {
				return after
			}
		}
		return val
	case "query":
		return c.Query(name)
	case "cookie":
		return c.Cookie(name)
	default:
		return ""
	}
}
