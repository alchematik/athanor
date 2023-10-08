package generator

import (
	"bytes"
	_ "embed"
	"go/format"
	"text/template"
)

//go:embed resource_op.tmpl
var resourceOp string

func (g Generator) GenerateResourceOp(resource Resource) ([]byte, error) {
	tmpl, err := template.New("resource_create").
		Funcs(template.FuncMap{}).Parse(resourceOp)
	if err != nil {
		return nil, err
	}

	imports := []string{
		"github.com/hashicorp/hcl/v2/gohcl",
		"github.com/hashicorp/hcl/v2",
	}

	tmplData := map[string]any{
		"Resource": resource,
		"Imports":  imports,
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
