package generator

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"text/template"
)

//go:embed resource.tmpl
var resourceTmpl string

func (g Generator) GenerateResource(r Resource) ([]byte, error) {
	tmpl, err := template.New("resource").
		Funcs(template.FuncMap{
			"convertIdentifierType": convertIdentifierType,
		}).
		Parse(resourceTmpl)
	if err != nil {
		return nil, err
	}

	imports := []string{
		"github.com/alchematik/athanor/provider",
	}
	tmplData := map[string]any{
		"Imports":  imports,
		"Resource": r,
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

func convertIdentifierType(part IdentifierPart) (string, error) {
	switch part.Type {
	case "string":
		return "string", nil
	case "identifier_oneof":
		return "identifier", nil
	default:
		return "", fmt.Errorf("unknown type: %s", part.Type)
	}

}
