package generator

import (
	"bytes"
	_ "embed"
	"go/format"
	"path/filepath"
	"text/template"
)

//go:embed provider.tmpl
var providerTmpl string

func (g Generator) GenerateProvider(s Schema) ([]byte, error) {
	tmpl, err := template.New("provider").
		Funcs(template.FuncMap{
			"toPascalCase": toPascalCase,
		}).
		Parse(providerTmpl)
	if err != nil {
		return nil, err
	}

	imports := []string{
		"fmt",
		"context",
		"github.com/alchematik/athanor/provider",
	}
	for _, r := range s.Resources {
		imports = append(imports, filepath.Join(g.ModName, g.ResourceDir, r.Name))
	}

	tmplData := map[string]any{
		"Schema":  s,
		"Imports": imports,
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
