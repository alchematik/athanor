package generator

import (
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"strings"
)

type Generator struct {
	ResourceDir string
	ModName     string
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

func identifierPartGoType(part IdentifierPart) (string, error) {
	switch part.Type {
	case "string":
		return "string", nil
	case "identifier_oneof":
		return "any", nil
	case "resource":
		return fmt.Sprintf("*%s.Identifier", part.Resource), nil
	default:
		return "", fmt.Errorf("unknown type: %q", part.Type)
	}
}

func configPartGoType(part ConfigPart) (string, error) {
	switch part.Type {
	case "string":
		return "string", nil
	case "identifier_oneof":
		return "any", nil
	default:
		return "", fmt.Errorf("unknown type: %q", part.Type)
	}
}
