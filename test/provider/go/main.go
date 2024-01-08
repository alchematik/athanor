package main

import (
	"fmt"

	"github.com/alchematik/athanor/sdk/go/provider/schema"
)

func main() {

	bucket := schema.ResourceSchema{
		Type: "bucket",
		Identifier: schema.FieldSchema{
			// Name: "identifier",
			Type: schema.FieldTypeIdentifier,
			Fields: []schema.FieldSchema{
				{
					Name: "account",
					Type: schema.FieldTypeString,
				},
				{
					Name: "region",
					Type: schema.FieldTypeString,
				},
				{
					Name: "name",
					Type: schema.FieldTypeString,
				},
			},
		},
		Config: schema.FieldSchema{
			Type: schema.FieldTypeStruct,
			Fields: []schema.FieldSchema{
				{
					Name: "expiration",
					Type: schema.FieldTypeString,
				},
			},
		},
		Attrs: schema.FieldSchema{
			Type: schema.FieldTypeStruct,
			Fields: []schema.FieldSchema{
				{
					Name: "bar",
					Type: schema.FieldTypeStruct,
					Fields: []schema.FieldSchema{
						{
							Name: "foo",
							Type: schema.FieldTypeString,
						},
					},
				},
			},
		},
	}

	bucketObject := schema.ResourceSchema{
		Type: "bucket_object",
		Identifier: schema.FieldSchema{
			Type: schema.FieldTypeStruct,
			Fields: []schema.FieldSchema{
				{
					Name:   "bucket",
					Type:   schema.FieldTypeIdentifier,
					Fields: bucket.Identifier.Fields,
				},
			},
		},
		Config: schema.FieldSchema{
			Type: schema.FieldTypeStruct,
			Fields: []schema.FieldSchema{
				{
					Name: "contents",
					Type: schema.FieldTypeString,
				},
				{
					Name: "some_field",
					Type: schema.FieldTypeString,
				},
			},
		},
		Attrs: schema.FieldSchema{},
	}

	s := schema.Schema{
		Name:    "gcp",
		Version: "v0.0.1",
		Resources: []schema.ResourceSchema{
			bucket,
			bucketObject,
		},
	}

	p := s.ToProto()
	fmt.Printf("proto: %v\n", p)

}
