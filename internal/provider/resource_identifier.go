package generator

import (
	"bytes"
	_ "embed"
	// "fmt"
	"go/format"
	"path/filepath"
	"text/template"
)

//go:embed resource_identifier.tmpl
var resourceIdentifier string

func (g Generator) GenerateResourceIdentifier(provider Provider, resource Resource) ([]byte, error) {
	tmpl, err := template.New("resource_identifier").
		Funcs(template.FuncMap{
			"toPascalCase":         toPascalCase,
			"identifierPartGoType": identifierPartGoType,
		}).
		Parse(resourceIdentifier)
	if err != nil {
		return nil, err
	}

	imports := []string{
		"fmt",
		"strings",
	}
	var metadata []IdentifierPart
	for _, id := range resource.Identifier {
		if id.Type == "resource" {
			imports = append(imports, filepath.Join(g.ModName, g.ResourceDir, id.Resource))
		}
	}

	tmplData := map[string]any{
		"Provider": provider,
		"Resource": resource,
		"Imports":  imports,
		"Metadata": metadata,
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
