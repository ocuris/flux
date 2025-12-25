package templates

import (
	"embed"
	"text/template"
)

//go:embed *.html
var TemplateFS embed.FS

var Tmpl = template.Must(template.ParseFS(TemplateFS, "*.html"))
