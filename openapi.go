package flux

// OpenAPISpec represents the OpenAPI specification
type OpenAPISpec struct {
	config Config
	paths  map[string]map[string]*any
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
