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

func backtick() string {
	return "`"
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

func identifierPartHCLGoType(part IdentifierPart) (string, error) {
	switch part.Type {
	case "string":
		return "string", nil
	case "identifier_oneof":
		return "identifier.HCLIdentifier", nil
	case "resource":
		return fmt.Sprintf("*%s.HCLIdentifier", part.Resource), nil
	default:
		return "", fmt.Errorf("unknown type: %q", part.Type)
	}
}

func identifierPartCtyType(part IdentifierPart) (string, error) {
	switch part.Type {
	case "string":
		return "cty.String", nil
	case "identifier_oneof":
		return fmt.Sprintf("id.%s.CtyType()", toPascalCase(part.Name)), nil
	case "resource":
		return fmt.Sprintf("id.%s.CtyType()", toPascalCase(part.Name)), nil
	default:
		return "", fmt.Errorf("unknown type: %q", part.Type)
	}
}

func identifierPartToIdentifier(part IdentifierPart) (string, error) {
	switch part.Type {
	case "string":
		return fmt.Sprintf("id.%s", toPascalCase(part.Name)), nil
	case "identifier_oneof":
		return fmt.Sprintf("id.%s.CtyType()", toPascalCase(part.Name)), nil
	case "resource":
		return fmt.Sprintf("id.%s.ToIdentifier()", toPascalCase(part.Name)), nil
	default:
		return "", fmt.Errorf("unknown type: %q", part.Type)
	}
}
