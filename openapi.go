package flux


// RouteOptions holds route metadata for OpenAPI
type RouteOptions struct {
	Summary     string
	Description string
	Tags        []string
	RequestBody interface{}
	Responses   map[int]interface{}
}

// RouteOption is a function that configures RouteOptions
type RouteOption func(*RouteOptions)
