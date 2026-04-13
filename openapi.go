package flux

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/ocuris/flux/templates"
)

// OpenAPISpec represents the OpenAPI specification
type OpenAPISpec struct {
	config   Config
	paths    map[string]map[string]*any
	specJSON []byte // pre-generated and cached
}

// RouteOptions holds route metadata for OpenAPI
type RouteOptions struct {
	Summary     string
	Description string
	Tags        []string
	RequestBody any
	Responses   map[int]any
}

// RouteOption is a function that configures RouteOptions
type RouteOption func(*RouteOptions)

// pathParamRe matches Flux-style path params like :id or :userID.
var pathParamRe = regexp.MustCompile(`:([^/]+)`)

// toOpenAPIPath converts a Flux route path (/users/:id) to OpenAPI 3.0 format
// (/users/{id}). Swagger UI requires curly-brace notation to render path
// parameter input fields correctly.
func toOpenAPIPath(fluxPath string) string {
	return pathParamRe.ReplaceAllString(fluxPath, "{$1}")
}

// InitOpenAPI registers the /docs and /openapi.json endpoints.
func (f *Flux) InitOpenAPI() {
	f.openapi = &OpenAPISpec{
		config: f.config,
		paths:  make(map[string]map[string]*any),
	}

	f.generateOpenAPISpec()

	f.GET("/docs", f.handleDocs)
	f.GET("/openapi.json", f.handleOpenAPIJSON)
}

func (f *Flux) generateOpenAPISpec() {
	paths := make(map[string]map[string]any)

	f.routesMu.RLock()
	for _, route := range f.registeredRoutes {
		if route.Path == "/docs" || route.Path == "/openapi.json" || route.Path == "/redoc" {
			continue
		}

		openAPIPath := toOpenAPIPath(route.Path)
		if _, exists := paths[openAPIPath]; !exists {
			paths[openAPIPath] = make(map[string]any)
		}

		methodKey := strings.ToLower(route.Method)
		var operation map[string]any
		if route.Doc != nil {
			operation = route.Doc.ToMap()
		} else {
			operation = map[string]any{
				"summary":     fmt.Sprintf("%s %s", route.Method, route.Path),
				"operationId": fmt.Sprintf("%s_%s", methodKey, openAPIPath),
				"responses": map[string]any{
					"200": map[string]any{
						"description": "Successful response",
					},
				},
			}
		}
		paths[openAPIPath][methodKey] = operation
	}
	f.routesMu.RUnlock()

	spec := map[string]any{
		"openapi": "3.0.0",
		"info": map[string]any{
			"title":       f.config.Title,
			"description": f.config.Description,
			"version":     f.config.Version,
		},
		"paths": paths,
	}

	data, _ := json.MarshalIndent(spec, "", "  ")
	f.openapi.specJSON = data
}

// handleDocs serves the Swagger UI documentation page.
func (f *Flux) handleDocs(c *Context) error {
	var buf bytes.Buffer
	if err := templates.Tmpl.ExecuteTemplate(&buf, "new.html", nil); err != nil {
		return fmt.Errorf("failed to render docs template: %w", err)
	}
	c.SetHeader("Cache-Control", "no-cache, no-store, must-revalidate")
	c.SetHeader("Pragma", "no-cache")
	c.SetHeader("Expires", "0")
	return c.HTML(http.StatusOK, buf.String())
}

// handleOpenAPIJSON generates and serves the OpenAPI 3.0 specification JSON.
//
// Path parameters are converted from Flux notation (:id) to OpenAPI notation
// ({id}) so Swagger UI renders interactive input fields for them.
// handleOpenAPIJSON serves the pre-generated OpenAPI spec.
func (f *Flux) handleOpenAPIJSON(c *Context) error {
	c.SetHeader("Cache-Control", "no-cache, no-store, must-revalidate")
	c.SetHeader("Pragma", "no-cache")
	c.SetHeader("Expires", "0")
	c.SetHeader("Content-Type", "application/json")
	_, err := c.Writer.Write(f.openapi.specJSON)
	return err
}
