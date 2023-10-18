package bucket

import (
	"github.com/alchematik/athanor/provider"
)

func ResourceSchema() provider.ResourceSchema {
	return provider.ResourceSchema{
		IdentifierFields: []provider.Field{

			{
				Name: "project",
				Type: "string",
			},

			{
				Name: "region",
				Type: "string",
			},

			{
				Name: "name",
				Type: "string",
			},
		},
		ConfigFields: []provider.Field{},
		DependsOn:    []string{},
	}
}
