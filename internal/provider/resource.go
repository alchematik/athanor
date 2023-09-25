package generator

import (
	"bytes"
	_ "embed"
	"go/format"
	"text/template"
)

//go:embed resource.tmpl
var resourceTmpl string

func (g Generator) GenerateResource(provider string, r Resource) ([]byte, error) {
	tmpl, err := template.New("resource").
		Funcs(template.FuncMap{}).
		Parse(resourceTmpl)
	if err != nil {
		return nil, err
	}

	tmplData := map[string]any{}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, tmplData); err != nil {
		return nil, err
	}

	src, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, err
	}

	return src, nil
}
