package generator

import (
	"bytes"
	_ "embed"
	"go/format"
	"text/template"
)

//go:embed client.tmpl
var clientTmpl string

func (g Generator) GenerateClient(resource Resource) ([]byte, error) {
	tmpl, err := template.New("client").
		Funcs(template.FuncMap{
			"toPascalCase": toPascalCase,
		}).
		Parse(clientTmpl)
	if err != nil {
		return nil, err
	}

	imports := []string{
		"context",
		"errors",
		"github.com/alchematik/athanor/provider",
	}

	tmplData := map[string]any{
		"Imports":  imports,
		"Resource": resource,
	}
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
