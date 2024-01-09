package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"strings"
	"text/template"

	providerpb "github.com/alchematik/athanor/internal/gen/go/proto/provider/v1"
)

//go:embed resource_header.tmpl
var resourceHeaderTmpl string

//go:embed struct_type.tmpl
var structTypeTmpl string

//go:embed string_type.tmpl
var stringTypeTmpl string

/*

- Generate resource ID struct
- Generate resource config struct
- Generate resource attrs struct
- Generate other struct fields

*/

func GenerateResourceType(name string, types []*providerpb.FieldSchema) ([]byte, error) {
	var out []byte

	tmpl, err := template.New("resource_header").Parse(resourceHeaderTmpl)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"PackageName": "provider",
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	out = append(out, buf.Bytes()...)

	for _, t := range types {
		switch t.GetType() {
		case providerpb.FieldType_STRING:
		case providerpb.FieldType_STRUCT:
			tmpl, err := template.New("struct_type").
				Funcs(template.FuncMap{
					"toPascalCase": toPascalCase,
					"toType": func(f *providerpb.FieldSchema) (string, error) {
						if f.GetIsIdentifier() {
							return "sdk.IdentifierType", nil
						}

						switch f.GetType() {
						case providerpb.FieldType_STRING:
							return "sdk.StringType", nil
						case providerpb.FieldType_STRUCT:
							return toPascalCase(f.GetName()), nil
						default:
							return "", fmt.Errorf("unrecognized type: %s", f.GetType())
						}
					},
				}).
				Parse(structTypeTmpl)
			if err != nil {
				return nil, err
			}

			data := map[string]any{
				"Name": t.GetName(),
				"Type": t,
			}

			var buffer bytes.Buffer
			if err := tmpl.Execute(&buffer, data); err != nil {
				return nil, err
			}

			out = append(out, buffer.Bytes()...)
		}
	}

	src, err := format.Source(out)
	if err != nil {
		return nil, err
	}

	return src, nil
}

func toPascalCase(str string) string {
	titleCaser := cases.Title(language.Und)
	upperCaser := cases.Upper(language.Und)
	splitter := func(r rune) bool {
		return r == '_' || r == ' '
	}
	parts := strings.FieldsFunc(str, splitter)
	if len(parts) == 1 {
		part := parts[0]
		if upperCaser.String(part) == part {
			return part
		}

		return titleCaser.String(parts[0])
	}

	var transformed []string
	for _, part := range parts {
		if upperCaser.String(part) == part {
			transformed = append(transformed, part)
			continue
		}

		transformed = append(transformed, titleCaser.String(part))
	}

	return strings.Join(transformed, "")
}

func findStructFields(field *providerpb.FieldSchema) []*providerpb.FieldSchema {
	// Use map?
	// Put all structs across resources in same package?
	var structFields []*providerpb.FieldSchema
	for _, f := range field.GetFields() {
		if f.GetType() == providerpb.FieldType_STRUCT {
			structFields = append(structFields, f)
		}
	}

	return structFields
}
