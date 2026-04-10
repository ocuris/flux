package flux

import "strconv"

// DocBuilder manages OpenAPI metadata for a route.
type DocBuilder struct {
	summary     string
	description string
	tags        []string
	parameters  []Parameter
	requestBody *RequestBodyDoc
	responses   map[int]*ResponseDoc
	security    []string
	metadata    map[string]interface{}
}

type Info struct {
	Summary     string
	Description string
	Tags        []string
}

// Param allows chaining parameters directly on an Info struct.
func (i Info) Param(name, in, description, schemaType string, required bool) *DocBuilder {
	return Doc(i.Summary, i.Description, i.Tags...).Param(name, in, description, schemaType, required)
}

// Response allows chaining responses directly on an Info struct.
func (i Info) Response(code int, description, contentType string, schema interface{}) *DocBuilder {
	return Doc(i.Summary, i.Description, i.Tags...).Response(code, description, contentType, schema)
}

// RequestBody allows chaining a request body schema directly on an Info struct.
func (i Info) RequestBody(description string, schema interface{}) *DocBuilder {
	return Doc(i.Summary, i.Description, i.Tags...).RequestBody(description, schema)
}


// Parameter describes a route parameter (path, query, header, etc.)
type Parameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // path, query, header, cookie
	Description string      `json:"description"`
	SchemaType  string      `json:"type"` // string, number, integer, boolean, array, object
	Required    bool        `json:"required"`
	Schema      *Schema     `json:"schema,omitempty"`
	Example     interface{} `json:"example,omitempty"`
}

// RequestBodyDoc describes request body
type RequestBodyDoc struct {
	Description string      `json:"description"`
	Schema      *Schema     `json:"schema"`
	Required    bool        `json:"required"`
	Example     interface{} `json:"example"`
}

// ResponseDoc describes a response
type ResponseDoc struct {
	Description string      `json:"description"`
	ContentType string      `json:"content_type"`
	Schema      *Schema     `json:"schema"`
	Example     interface{} `json:"example"`
}

// Schema describes OpenAPI schema
type Schema struct {
	Type        string             `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Example     interface{}        `json:"example,omitempty"`
	Default     interface{}        `json:"default,omitempty"`
	Enum        []interface{}      `json:"enum,omitempty"`
	MinLength   *int               `json:"minLength,omitempty"`
	MaxLength   *int               `json:"maxLength,omitempty"`
	Pattern     string             `json:"pattern,omitempty"`
	Minimum     *float64           `json:"minimum,omitempty"`
	Maximum     *float64           `json:"maximum,omitempty"`
	Required    []string           `json:"required,omitempty"`
}

// Doc starts a new DocBuilder with positional arguments.
func Doc(summary, description string, tags ...string) *DocBuilder {
	return &DocBuilder{
		summary:     summary,
		description: description,
		tags:        tags,
		parameters:  make([]Parameter, 0),
		responses:   make(map[int]*ResponseDoc),
		security:    make([]string, 0),
		metadata:    make(map[string]interface{}),
	}
}

// Summary sets the summary
func (d *DocBuilder) Summary(s string) *DocBuilder {
	d.summary = s
	return d
}

// Description sets the description
func (d *DocBuilder) Description(desc string) *DocBuilder {
	d.description = desc
	return d
}

// Tags sets tags for grouping
func (d *DocBuilder) Tags(t ...string) *DocBuilder {
	d.tags = t
	return d
}

// Param adds a parameter (path, query, header, cookie)
func (d *DocBuilder) Param(name, in, description, schemaType string, required bool) *DocBuilder {
	d.parameters = append(d.parameters, Parameter{
		Name:        name,
		In:          in,
		Description: description,
		SchemaType:  schemaType,
		Required:    required,
	})
	return d
}

// ParamWithExample adds a parameter with example value
func (d *DocBuilder) ParamWithExample(name, in, description, schemaType string, required bool, example interface{}) *DocBuilder {
	d.parameters = append(d.parameters, Parameter{
		Name:        name,
		In:          in,
		Description: description,
		SchemaType:  schemaType,
		Required:    required,
		Example:     example,
	})
	return d
}

// RequestBody sets the request body documentation
func (d *DocBuilder) RequestBody(description string, example interface{}) *DocBuilder {
	d.requestBody = &RequestBodyDoc{
		Description: description,
		Required:    true,
		Example:     example,
	}
	return d
}

// Response adds response documentation
func (d *DocBuilder) Response(statusCode int, description, contentType string, example interface{}) *DocBuilder {
	d.responses[statusCode] = &ResponseDoc{
		Description: description,
		ContentType: contentType,
		Example:     example,
	}
	return d
}

// Security adds security requirement
func (d *DocBuilder) Security(scheme string) *DocBuilder {
	d.security = append(d.security, scheme)
	return d
}

// Meta adds arbitrary metadata
func (d *DocBuilder) Meta(key string, value interface{}) *DocBuilder {
	d.metadata[key] = value
	return d
}

// Deprecated marks endpoint as deprecated
func (d *DocBuilder) Deprecated(b bool) *DocBuilder {
	d.metadata["deprecated"] = b
	return d
}

// OperationID sets OpenAPI operationId
func (d *DocBuilder) OperationID(id string) *DocBuilder {
	d.metadata["operationId"] = id
	return d
}

// ToMap converts to map for OpenAPI spec
func (d *DocBuilder) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"summary":     d.summary,
		"description": d.description,
	}

	// Add tags if any
	if len(d.tags) > 0 {
		result["tags"] = d.tags
	}

	// Add parameters if any
	if len(d.parameters) > 0 {
		params := make([]map[string]interface{}, len(d.parameters))
		for i, p := range d.parameters {
			params[i] = map[string]interface{}{
				"name":        p.Name,
				"in":          p.In,
				"description": p.Description,
				"required":    p.Required,
				"schema": map[string]interface{}{
					"type": p.SchemaType,
				},
			}
			if p.Example != nil {
				params[i]["example"] = p.Example
			}
		}
		result["parameters"] = params
	}

	// Add request body if any
	if d.requestBody != nil {
		result["requestBody"] = map[string]interface{}{
			"description": d.requestBody.Description,
			"required":    d.requestBody.Required,
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
					},
				},
			},
		}
		if d.requestBody.Example != nil {
			result["requestBody"].(map[string]interface{})["content"].(map[string]interface{})["application/json"].(map[string]interface{})["example"] = d.requestBody.Example
		}
	}

	// Add responses
	responses := make(map[string]interface{})
	for statusCode, resp := range d.responses {
		statusStr := statusCodeToString(statusCode)
		responses[statusStr] = map[string]interface{}{
			"description": resp.Description,
		}

		if resp.ContentType != "" {
			contentNode := map[string]interface{}{
				"schema": map[string]interface{}{
					"type": "object",
				},
			}
			if resp.Example != nil {
				contentNode["example"] = resp.Example
			}
			responses[statusStr].(map[string]interface{})["content"] = map[string]interface{}{
				resp.ContentType: contentNode,
			}
		}
	}
	result["responses"] = responses

	// Add metadata
	for k, v := range d.metadata {
		result[k] = v
	}

	return result
}

// statusCodeToString converts an HTTP status code integer to its string
// representation for use as an OpenAPI response key.
func statusCodeToString(code int) string {
	return strconv.Itoa(code)
}
