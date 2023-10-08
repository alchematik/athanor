package generator

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"path/filepath"
	"text/template"
)

//go:embed resource_identifier.tmpl
var resourceIdentifier string

func (g Generator) GenerateResourceIdentifier(provider Provider, resource Resource) ([]byte, error) {
	tmpl, err := template.New("resource_identifier").
		Funcs(template.FuncMap{
			"toPascalCase":            toPascalCase,
			"identifierPartGoType":    identifierPartGoType,
			"identifierPartHCLGoType": identifierPartHCLGoType,
			"identifierPartCtyType":   identifierPartCtyType,
		}).
		Parse(resourceIdentifier)
	if err != nil {
		return nil, err
	}

	imports := []string{
		"fmt",
		"strings",
		"github.com/zclconf/go-cty/cty",
		"github.com/zclconf/go-cty/cty/gocty",
		"github.com/hashicorp/hcl/v2",
		"github.com/hashicorp/hcl/v2/gohcl",
		"github.com/alchematik/athanor/identifier",
	}
	var metadata []IdentifierPart
	for _, id := range resource.Identifier {
		if id.Type == "resource" {
			imports = append(imports, filepath.Join(g.ModName, g.ResourceDir, id.Resource))
		}
		if id.Type == "identifier_oneof" {
			for _, choice := range id.Choices {
				imports = append(imports, filepath.Join(g.ModName, g.ResourceDir, choice))
			}
			metadata = append(metadata, IdentifierPart{
				Name: fmt.Sprintf("%s_type", id.Name),
				Type: "string",
			})
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
