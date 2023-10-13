package generator

import (
	"bytes"
	_ "embed"
	"github.com/dominikbraun/graph"
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
		"github.com/hashicorp/hcl/v2",
		"github.com/alchematik/athanor/provider",
	}
	dag := graph.New(graph.StringHash, graph.Directed(), graph.Acyclic(), graph.PreventCycles())
	if err := dag.AddVertex("root"); err != nil {
		return nil, err
	}
	for _, r := range s.Resources {
		imports = append(imports, filepath.Join(g.ModName, g.ResourceDir, r.Name))
		if err := dag.AddVertex(r.Name); err != nil {
			return nil, err
		}

		deps := r.Dependencies()
		if len(deps) == 0 {
			if err := dag.AddEdge("root", r.Name); err != nil {
				return nil, err
			}
			continue
		}

		for _, dep := range deps {
			if err := dag.AddEdge(dep, r.Name); err != nil {
				return nil, err
			}
		}
	}

	resources, err := graph.TopologicalSort(dag)
	if err != nil {
		return nil, err
	}

	// Remove root node
	resources = resources[1:]

	tmplData := map[string]any{
		"Schema":        s,
		"Imports":       imports,
		"ResourceNames": resources,
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
