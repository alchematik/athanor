package generator

import (
	"github.com/hashicorp/hcl/v2/hclsimple"
)

type Parser struct {
}

func (p Parser) Parse(name string, data []byte) (Schema, error) {
	var s Schema
	if err := hclsimple.Decode(name, data, nil, &s); err != nil {
		return Schema{}, err
	}

	return s, nil
}

type Schema struct {
	Provider  Provider   `hcl:"provider,block"`
	Resources []Resource `hcl:"resource,block"`
}

type Provider struct {
	Version string `hcl:"version"`
	Name    string `hcl:"name"`
}

type IdentifierPart struct {
	Name        string   `hcl:"name,label"`
	Type        string   `hcl:"type"`
	Description string   `hcl:"description"`
	IsNamed     bool     `hcl:"is_named,optional"`
	Choices     []string `hcl:"choices,optional"`
	Resource    string   `hcl:"resource,optional"`
}

type ConfigPart struct {
	Name        string   `hcl:"name,label"`
	Type        string   `hcl:"type"`
	Description string   `hcl:"description"`
	Immutable   bool     `hcl:"immutable,optional"`
	Key         string   `hcl:"key,optional"`
	Value       string   `hcl:"value,optional"`
	Choices     []string `hcl:"choices,optional"`
}

type Resource struct {
	Name       string           `hcl:"name,label"`
	Modifiers  []string         `hcl:"modifiers"`
	Identifier []IdentifierPart `hcl:"identifier,block"`
	Config     []ConfigPart     `hcl:"config,block"`
}

func (r Resource) Dependencies() []string {
	var deps []string
	for _, id := range r.Identifier {
		switch id.Type {
		case "resource":
			deps = append(deps, id.Name)
		case "identifier_oneof":
			for _, c := range id.Choices {
				deps = append(deps, c)
			}
		}
	}

	return deps
}

func (r Resource) HasCreate() bool {
	for _, m := range r.Modifiers {
		if m == "create" {
			return true
		}
	}

	return false
}
