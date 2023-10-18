package bucket_object

import (
	"github.com/alchematik/athanor/provider"
)

func ResourceSchema() provider.ResourceSchema {
	return provider.ResourceSchema{
		IdentifierFields: []provider.Field{

			{
				Name: "bucket",
				Type: "bucket",
			},

			{
				Name: "name",
				Type: "string",
			},
		},
		ConfigFields: []provider.Field{

			{
				Name: "contents",
				Type: "string",
			},
		},
		DependsOn: []string{

			"bucket",
		},
	}
}
