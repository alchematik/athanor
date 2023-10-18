package resource_policy

import (
	"github.com/alchematik/athanor/provider"
)

func ResourceSchema() provider.ResourceSchema {
	return provider.ResourceSchema{
		IdentifierFields: []provider.Field{

			{
				Name: "resource",
				Type: "identifier",
			},

			{
				Name: "name",
				Type: "string",
			},
		},
		ConfigFields: []provider.Field{},
		DependsOn: []string{

			"bucket",
		},
	}
}
