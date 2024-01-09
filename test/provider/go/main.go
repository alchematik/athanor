package main

import (
	"encoding/json"
	"os"

	"github.com/alchematik/athanor/sdk/go/provider/schema"
)

func main() {

	bucket := schema.ResourceSchema{
		Type: "bucket",
		Identifier: schema.FieldSchema{
			Name:         "identifier",
			Type:         schema.FieldTypeStruct,
			IsIdentifier: true,
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
			Name: "config",
			Type: schema.FieldTypeStruct,
			Fields: []schema.FieldSchema{
				{
					Name: "expiration",
					Type: schema.FieldTypeString,
				},
			},
		},
		Attrs: schema.FieldSchema{
			Name: "attrs",
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
			Name: "identifier",
			Type: schema.FieldTypeStruct,
			Fields: []schema.FieldSchema{
				{
					Name:         "bucket",
					Type:         schema.FieldTypeStruct,
					Fields:       bucket.Identifier.Fields,
					IsIdentifier: true,
				},
			},
		},
		Config: schema.FieldSchema{
			Name: "config",
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
		Attrs: schema.FieldSchema{
			Name: "attrs",
			Type: schema.FieldTypeStruct,
		},
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

	outputPath := os.Args[1]

	f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {

		os.Exit(1)
	}

	data, err := json.Marshal(p)
	if err != nil {
		os.Exit(1)
	}

	if _, err := f.Write(data); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
